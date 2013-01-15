package process

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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
	NoCogCode = errors.New("No generator code found in file")
)

func Cog(file string, opt *Options) error {
	d := &data{file, opt}
	return d.cog()
}

type data struct {
	File string
	Opt  *Options
}

func (d *data) cog() error {
	log.Printf("Processing file '%s'", d.File)

	output, err := d.tryCog()

	if err == NoCogCode {
		os.Remove(output)
		log.Printf("%s '%s'", err, d.File)
		return err
	}
	// this is the success case - got to the end of the file without any other errors
	if err == io.EOF {
		/*
			if err := os.Remove(d.File); err != nil {
				log.Printf("Error removing original file '%s': %s", d.File, err)
				return err
			}
			if err := os.Rename(output, d.File); err != nil {
				log.Printf("Error renaming cog file '%s': %s", output, err)
				return err
			}
		*/
		log.Printf("Successfully processed '%s'", d.File)
		return nil
	} else {
		log.Printf("Error processing cog file '%s': %s", d.File, err)
		if output != "" {
			if err := os.Remove(output); err != nil {
				log.Printf("Error removing partial cog file '%s': %s", output, err)
			}
		}
		return err
	}
	return nil
}

func (d *data) tryCog() (string, error) {
	in, err := os.Open(d.File)
	if err != nil {
		log.Printf("Error reading file '%s': %s", d.File, err)
		return "", err
	}
	r := bufio.NewReader(in)

	defer in.Close()

	out := d.File + "_cog"
	log.Printf("Writing output to %s\n", out)
	f, err := createNew(out)
	if err != nil {
		return "", fmt.Errorf("Error creating cog output file: %s", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	return out, d.gen(r, w)
}

func (d *data) gen(r *bufio.Reader, w *bufio.Writer) error {
	firstRun := true
	for {
		if err := cogPlainText(r, w, firstRun); err != nil {
			return err
		}
		firstRun = false

		if err := cogGeneratorCode(r, w, d.File); err != nil {
			return err
		}

		if err := cogToEnd(r, w, d.Opt.UseEOF); err != nil {
			return err
		}
	}
	panic("Can't get here!")
}

func cogPlainText(r *bufio.Reader, w *bufio.Writer, firstRun bool) error {
	log.Println("cogging plaintext")
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
	log.Printf("Wrote %d lines to output file\n", len(lines))
	return err
}

func cogGeneratorCode(r *bufio.Reader, w *bufio.Writer, name string) error {
	log.Println("cogging generator code")
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
	log.Printf("Wrote %d lines to output file\n", len(lines))

	// todo: handle other languages
	gen := fmt.Sprintf("%s_cog_%s", name, ".go")

	// write all but the last line to the generator file
	if err := writeAllNew(gen, lines[:len(lines)-1]); err != nil {
		return fmt.Errorf("Error writing code generation file: %s", err)
	}
	log.Printf("Wrote %d lines to generator file\n", len(lines)-1)

	//defer os.Remove(gen)
	if err := generate(w, gen); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

func cogToEnd(r *bufio.Reader, w *bufio.Writer, useEOF bool) error {
	log.Println("cogging to end")
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

	log.Println("Wrote 1 line to output file")

	// return original error from findLine
	// that way we return EOF if we get to the end of the file
	return err
}
func generate(w *bufio.Writer, name string) error {
	cmd := exec.Command("go", "run", name)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
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
				return fmt.Errorf("Error writing to and closing newfile %s: %s\n%s", name, err, err2)
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
			log.Printf("Found marker %s after %d lines", marker, len(lines))
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
