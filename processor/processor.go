// Package processor contains the code to generate text from embedded sourcecode.
package processor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	// Indicates a file was processed, but no gocog markers were found in it
	NoCogCode = errors.New("NoCogCode")

	// Indicates a malformed gocog section, missing either the GOCOG_END or END statements
	UnexpectedEOF = errors.New("UnexpectedEOF")
)

// Process the given file with the given options
func Run(file string, opt *Options) error {
	if opt == nil {
		opt = &Options{}
	}

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
				d.log.Println(err)
			}
		}
		return err
	}
	return nil
}

func (d *data) tryCog() (output string, err error) {
	in, err := os.Open(d.File)
	if err != nil {
		return "", err
	}
	defer in.Close()

	r := bufio.NewReader(in)

	output = d.File + "_cog"
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
		prefix, err := d.cogPlainText(r, w, firstRun)
		if err != nil {
			return err
		}
		firstRun = false

		if err := d.cogGeneratorCode(r, w, d.File, prefix); err != nil {
			return err
		}

		if err := d.cogToEnd(r, w); err != nil {
			return err
		}
	}
	panic("Can't get here!")
}

func (d *data) cogPlainText(r *bufio.Reader, w io.Writer, firstRun bool) (prefix string, err error) {
	d.Tracef("cogging plaintext")
	mark := d.Opt.StartMark + "gocog"
	lines, found, err := readUntil(r, mark)
	if err == io.EOF {
		if found {
			// found gocog statement, but nothing after it
			return "", UnexpectedEOF
		}
		if firstRun {
			// default case - no cog code, don't bother to write out anything
			return "", NoCogCode
		}
		// didn't find it, but this isn't the first time we've run
		// so no big deal, we just ran off the end of the file.
	}
	if err != nil && err != io.EOF {
		return "", err
	}

	// we can just write out the non-cog code to the output file
	// this also writes out the cog start line (if any)
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return "", err
		}
	}
	d.Tracef("Wrote %d lines to output file", len(lines))

	if !found {
		return "", err
	}

	return getPrefix(lines, mark), err
}

// Reads lines from the reader until reaching the gocog endmark
// Writes out the generator code to a file with the given name
// any lines that start with whitespace and then prefix will have
// prefix replaced by an equal number of spaces
func (d *data) cogGeneratorCode(r *bufio.Reader, w io.Writer, name, prefix string) error {
	d.Tracef("cogging generator code")
	lines, _, err := readUntil(r, "gocog"+d.Opt.EndMark)
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

	if !d.Opt.Excise {
		gen := fmt.Sprintf("%s_cog_%s", name, d.Opt.Ext)
		defer os.Remove(gen)

		// write all but the last line to the generator file
		if err := writeNewFile(gen, lines[:len(lines)-1], prefix); err != nil {
			return err
		}

		if err := d.runGen(gen, w); err != nil {
			return err
		}
	}

	return nil
}

func (d *data) runGen(f string, w io.Writer) error {
	cmd := d.Opt.Command
	if strings.Contains(cmd, "%s") {
		cmd = fmt.Sprintf(cmd, f)
	}
	args := make([]string, len(d.Opt.Args), len(d.Opt.Args))
	for i, s := range d.Opt.Args {
		if strings.Contains(s, "%s") {
			args[i] = fmt.Sprintf(s, f)
		} else {
			args[i] = s
		}
	}

	if err := run(cmd, args, w, d.log); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

func (d *data) cogToEnd(r *bufio.Reader, w io.Writer) error {
	d.Tracef("cogging to end")
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, found, err := findLine(r, d.Opt.StartMark+"end"+d.Opt.EndMark)
	if err == io.EOF && !found {
		if !d.Opt.UseEOF {
			return UnexpectedEOF
		}
		d.Tracef("No gocog end statement, treating EOF as end statement.")
		return io.EOF
	}
	if err != nil && err != io.EOF {
		return err
	}

	// if there's no error, found should always be true, so just write out
	if _, err := w.Write([]byte(line)); err != nil {
		return err
	}
	d.Tracef("Wrote 1 line to output file")
	return err
}
