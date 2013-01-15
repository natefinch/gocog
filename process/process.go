package process

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	GOCOG_START = "[[[gocog"
	GOCOG_END   = "gocog]]]"
	END         = "[[[end]]]"
)

var (
	NoCogCode = errors.New("NoCogCode")
)

func Cog(file string, opt *Options) error {
	var logger *log.Logger
	switch len(opt.Verbose) {
	case 0:
		logger = log.New(ioutil.Discard, "", log.LstdFlags)
	default:
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	d := &data{file, opt, logger}
	return d.cog()
}

type data struct {
	File string
	Opt  *Options
	log  *log.Logger
}

func (d *data) Tracef(format string, v ...interface{}) {
	if len(d.Opt.Verbose) > 1 {
		d.log.Printf(format, v...)
	}
}

func (d *data) cog() error {
	d.log.Printf("Processing file '%s'", d.File)

	output, err := d.tryCog()
	d.Tracef("Output file: '%s'", output)

	if err == NoCogCode {
		if err := os.Remove(output); err != nil {
			d.log.Printf("Error removing temporary output file '%s': %s", output, err)
		}
		d.log.Printf("No generator code found in file '%s'", d.File)
		return err
	}

	// this is the success case - got to the end of the file without any other errors
	if err == io.EOF {
		if err := os.Remove(d.File); err != nil {
			d.log.Printf("Error removing original file '%s': %s", d.File, err)
			return err
		}
		d.Tracef("Renaming output file '%s' to original filename '%s'", output, d.File)
		if err := os.Rename(output, d.File); err != nil {
			d.log.Printf("Error renaming cog file '%s': %s", output, err)
			return err
		}
		d.log.Printf("Successfully processed '%s'", d.File)
		return nil
	} else {
		d.log.Printf("Error processing cog file '%s': %s", d.File, err)
		if output != "" {
			if err := os.Remove(output); err != nil {
				d.log.Printf("Error removing partial cog file '%s': %s", output, err)
			}
		}
		return err
	}
	return nil
}

func (d *data) tryCog() (string, error) {
	in, err := os.Open(d.File)
	if err != nil {
		d.log.Printf("Error reading file '%s': %s", d.File, err)
		return "", err
	}
	defer in.Close()

	r := bufio.NewReader(in)

	output := d.File + "_cog"
	d.Tracef("Writing output to %s", output)
	out, err := createNew(output)
	if err != nil {
		return "", fmt.Errorf("Error creating cog output file: %s", err)
	}
	defer out.Close()

	return output, d.gen(r, out)
}

func (d *data) gen(r *bufio.Reader, w io.Writer) error {
	firstRun := true
	for {
		if err := d.cogPlainText(r, w, firstRun); err != nil {
			return err
		}
		firstRun = false

		if err := d.cogGeneratorCode(r, w, d.File); err != nil {
			return err
		}

		if err := d.cogToEnd(r, w, d.Opt.UseEOF); err != nil {
			return err
		}
	}
	panic("Can't get here!")
}

func (d *data) cogPlainText(r *bufio.Reader, w io.Writer, firstRun bool) error {
	d.Tracef("cogging plaintext")
	lines, err := readUntil(r, GOCOG_START)
	if err == io.EOF && firstRun {
		// default case - no cog code, don't bother to write out anything
		return NoCogCode
	}

	// we can just write out the non-cog code to the output file
	// this also writes out the cog start line
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return fmt.Errorf("Error writing to output file: %s", err)
		}
	}
	d.Tracef("Wrote %d lines to output file", len(lines))
	return err
}

func (d *data) cogGeneratorCode(r *bufio.Reader, w io.Writer, name string) error {
	d.Tracef("cogging generator code")
	lines, err := readUntil(r, GOCOG_END)
	if err == io.EOF {
		return fmt.Errorf("Unexpected EOF while looking for generator code.")
	}
	if err != nil {
		return err
	}
	// we have to write this out both to the output file and to the code file that we'll be running
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return fmt.Errorf("Error writing to output file: %s", err)
		}
	}
	d.Tracef("Wrote %d lines to output file", len(lines))

	// todo: handle other languages
	gen := fmt.Sprintf("%s_cog_%s", name, ".go")

	// write all but the last line to the generator file
	if err := writeAllNew(gen, lines[:len(lines)-1]); err != nil {
		return fmt.Errorf("Error writing code generation file: %s", err)
	}
	defer os.Remove(gen)

	if err := run(gen, w, (*LogWriter)(d.log)); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

type LogWriter log.Logger

func (lw *LogWriter) Write(b []byte) (int, error) {
	(*log.Logger)(lw).Print(string(b))
	return len(b), nil
}

func (d *data) cogToEnd(r *bufio.Reader, w io.Writer, useEOF bool) error {
	d.Tracef("cogging to end")
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, err := findLine(r, END)
	if err == io.EOF && !useEOF {
		return fmt.Errorf("Unexpected EOF while looking for end of generated code.")
	}
	if err != nil && err != io.EOF {
		return err
	}

	_, err2 := w.Write([]byte(line))
	if err2 != nil {
		return fmt.Errorf("Error writing to output file: %s", err2)
	}

	d.Tracef("Wrote 1 line to output file")

	// return original error from findLine
	// that way we return EOF if we get to the end of the file
	return err
}

func run(name string, out io.Writer, err io.Writer) error {
	cmd := exec.Command("go", "run", name)
	cmd.Stdout = out
	cmd.Stderr = err
	return cmd.Run()
}

func writeAllNew(name string, lines []string) error {
	out, err := createNew(name)
	if err != nil {
		return err
	}

	for _, line := range lines {
		if _, err := out.Write([]byte(line)); err != nil {
			if err2 := out.Close(); err2 != nil {
				return fmt.Errorf("Error writing to and closing newfile %s: %s%s", name, err, err2)
			}
			return fmt.Errorf("Error writing to newfile %s: %s", name, err)
		}
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("Error closing newfile %s: %s", name, err)
	}
	return nil
}

func readUntil(r *bufio.Reader, marker string) ([]string, error) {
	lines := make([]string, 0, 50)
	var err error
	for err == nil {
		var line string
		line, err = r.ReadString('\n')
		if line != "" {
			lines = append(lines, line)
		}
		if strings.Contains(line, marker) {
			return lines, err
		}
	}
	return lines, err
}

func findLine(r *bufio.Reader, marker string) (string, error) {
	var err error
	for err == nil {
		line, err := r.ReadString('\n')
		if err == nil || err == io.EOF {
			if strings.Contains(line, marker) {
				return line, nil
			}
		}
	}
	return "", err
}

func createNew(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if os.IsExist(err) {
		return f, fmt.Errorf("File '%s' already exists.", filename)
	}
	return f, err
}
