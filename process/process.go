package process

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	START   = "[[[gocog"
	END     = "gocog]]]"
	COG_END = "[[[end]]]"
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

	if err == io.EOF {
		if output != "" {
			if err := os.Remove(d.File); err != nil {
				return fmt.Errorf("Error removing original file %s: %s", d.File, err)
			}
			if err := os.Rename(output, d.File); err != nil {
				return fmt.Errorf("Error renaming cog file %s: %s", output, err)
			}
			log.Printf("Skipping non-cog file: %s", d.File)
			return nil
		}
		if err := os.Remove(output); err != nil {
			return fmt.Errorf("Error removing empty cog file %s: %s", output, err)
		}
		return nil
	} else {
		if err := os.Remove(output); err != nil {
			log.Printf("Error removing partial cog file %s: %s", output, err)
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

	tmpFile := d.File + "_cog"

	output, err := d.writeOutput(tmpFile, r)
	if output {
		return tmpFile, err
	}
	return "", err
}

func (d *data) writeOutput(out string, r *bufio.Reader) (bool, error) {
	f, err := createNew(out)
	if err != nil {
		return false, fmt.Errorf("Error creating cog output file: %s", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	return d.gen(r, w)
}

func (d *data) gen(r *bufio.Reader, w *bufio.Writer) (bool, error) {
	firstRun := true
	for {
		if err := cogPlainText(r, w, firstRun); err != nil {
			return !firstRun, err
		}
		firstRun = false

		if err := cogGeneratorCode(r, w, d.File); err != nil {
			return true, err
		}

		if err := cogToEnd(r, w, d.Opt.UseEOF); err != nil {
			return true, err
		}
	}
	panic("Can't get here!")
}

func cogToEnd(r *bufio.Reader, w *bufio.Writer, useEOF bool) error {
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, err := findLine(r, COG_END)
	if err == io.EOF && !useEOF {
		return fmt.Errorf("Unexpected EOF while looking for end of generated code.")
	}
	if err != nil && err != io.EOF {
		return err
	}

	_, err = w.Write([]byte(line))
	if err != nil {
		return fmt.Errorf("Error writing to output file: %s", err)
	}
	return nil
}

func cogPlainText(r *bufio.Reader, w *bufio.Writer, firstRun bool) error {
	lines, err := readUntil(r, START)
	if err == io.EOF && firstRun {
		// default case - no cog code, don't bother to write out anything
		return err
	}

	// we can just write out the non-cog code to the output file
	// this also writes out the cog start line
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return fmt.Errorf("Error writing to output file: %s", err)
		}
	}

	return err
}

func cogGeneratorCode(r *bufio.Reader, w *bufio.Writer, name string) error {
	lines, err := readUntil(r, END)
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

	// todo: handle other languages
	gen := fmt.Sprintf("%s_cog_%s", name, ".go")

	// write all but the last line to the generator file
	if err := writeAllNew(gen, lines[:len(lines)-1]); err != nil {
		return fmt.Errorf("Error writing code generation file: %s", err)
	}
	defer os.Remove(gen)
	if err := generate(w, gen); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
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
