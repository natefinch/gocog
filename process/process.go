package process

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const (
	GOCOG_START = "[[[gocog"
	GOCOG_END   = "gocog]]]"
	END         = "[[[end]]]"
)

var (
	NoCogCode     = errors.New("No cog code found in file")
	UnexpectedEOF = errors.New("Unexpected EOF in file")
)

func Cog(file string, opt *Options, wg *sync.WaitGroup) error {
	defer wg.Done()
	var logger *log.Logger
	if opt.Quiet {
		logger = log.New(ioutil.Discard, "", log.LstdFlags)
	} else {
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
	if d.Opt.Verbose {
		d.log.Printf(format, v...)
	}
}

func (d *data) cog() error {
	d.Tracef("Processing file '%s'", d.File)

	output, err := d.tryCog()
	d.Tracef("Output file: '%s'", output)

	if err == NoCogCode {
		if err := os.Remove(output); err != nil {
			d.log.Println(err)
		}
		d.log.Printf("No generator code found in file '%s'", d.File)
		return err
	}

	// this is the success case - got to the end of the file without any other errors
	if err == io.EOF {
		if err := os.Remove(d.File); err != nil {
			d.log.Printf("Error removing original file: %s", d.File, err)
			return err
		}
		d.Tracef("Renaming output file '%s' to original filename '%s'", output, d.File)
		if err := os.Rename(output, d.File); err != nil {
			d.log.Printf("Error renaming cog file: %s", output, err)
			return err
		}
		d.log.Printf("Successfully processed '%s'", d.File)
		return nil
	} else {
		d.log.Printf("Error processing cog file '%s': %s", d.File, err)
		if output != "" {
			if err := os.Remove(output); err != nil {
				d.log.Println(err)
			}
		}
		return err
	}
	return nil
}

func (d *data) tryCog() (string, error) {
	in, err := os.Open(d.File)
	if err != nil {
		return "", err
	}
	defer in.Close()

	r := bufio.NewReader(in)

	output := d.File + "_cog"
	d.Tracef("Writing output to %s", output)
	out, err := createNew(output)
	if err != nil {
		return "", err
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
			return err
		}
	}
	d.Tracef("Wrote %d lines to output file", len(lines))
	return err
}

func (d *data) cogGeneratorCode(r *bufio.Reader, w io.Writer, name string) error {
	d.Tracef("cogging generator code")
	lines, err := readUntil(r, GOCOG_END)
	if err == io.EOF {
		return UnexpectedEOF
	}
	if err != nil {
		return err
	}
	// we have to write this out both to the output file and to the code file that we'll be running
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}
	d.Tracef("Wrote %d lines to output file", len(lines))

	// todo: handle other languages
	gen := fmt.Sprintf("%s_cog_%s", name, ".go")

	// write all but the last line to the generator file
	if err := writeNewFile(gen, lines[:len(lines)-1]); err != nil {
		return err
	}
	defer os.Remove(gen)

	if err := run(gen, w, d.log); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

func (d *data) cogToEnd(r *bufio.Reader, w io.Writer, useEOF bool) error {
	d.Tracef("cogging to end")
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, err := findLine(r, END)
	if err == io.EOF && !useEOF {
		return UnexpectedEOF
	}
	if err != nil && err != io.EOF {
		return err
	}

	if _, err := w.Write([]byte(line)); err != nil {
		return err
	}

	d.Tracef("Wrote 1 line to output file")

	// return original error from findLine
	// that way we return EOF if we get to the end of the file
	return err
}
