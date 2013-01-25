// Package processor contains the code to generate text from embedded sourcecode.
package processor

import (
	"bufio"
	"bytes"
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

	newline byte = 10
)

// Process the given file with the given options
// This will read the file, rewriting to a temporary file
// then run any embedded code, using the given options
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
	c := &context{file, opt, logger}
	return c.cog()
}

type context struct {
	File string
	Opt  *Options
	log  *log.Logger
}

func (c *context) Tracef(format string, v ...interface{}) {
	if c.Opt.Verbose {
		c.log.Printf(format, v...)
	}
}

func (c *context) cog() error {
	c.Tracef("Processing file '%s'", c.File)

	output, err := c.tryCog()
	c.Tracef("Output file: '%s'", output)

	if err == NoCogCode {
		if err := os.Remove(output); err != nil {
			c.log.Println(err)
		}
		c.log.Printf("No generator code found in file '%s'", c.File)
		return err
	}

	// this is the success case - got to the end of the file without any other errors
	if err == io.EOF {
		if err := os.Remove(c.File); err != nil {
			c.log.Printf("Error removing original file '%s': %s", c.File, err)
			return err
		}
		c.Tracef("Renaming output file '%s' to original filename '%s'", output, c.File)
		if err := os.Rename(output, c.File); err != nil {
			c.log.Printf("Error renaming cog file '%s': %s", output, err)
			return err
		}
		c.log.Printf("Successfully processed '%s'", c.File)
		return nil
	} else {
		c.log.Printf("Error processing cog file '%s': %s", c.File, err)
		if output != "" {
			if err := os.Remove(output); err != nil {
				c.log.Println(err)
			}
		}
		return err
	}
	return nil
}

func (c *context) tryCog() (output string, err error) {
	in, err := os.Open(c.File)
	if err != nil {
		return "", err
	}
	defer in.Close()

	r := bufio.NewReader(in)

	output = c.File + "_cog"
	c.Tracef("Writing output to %s", output)
	out, err := createNew(output)
	if err != nil {
		return "", err
	}
	defer out.Close()

	return output, c.gen(r, out)
}

func (c *context) gen(r *bufio.Reader, w io.Writer) error {
	firstRun := true
	for {
		prefix, err := c.cogPlainText(r, w, firstRun)
		if err != nil {
			return err
		}
		firstRun = false

		if err := c.cogGeneratorCode(r, w, prefix); err != nil {
			return err
		}

		if err := c.cogToEnd(r, w); err != nil {
			return err
		}
	}
	panic("Can't get here!")
}

func (c *context) cogPlainText(r *bufio.Reader, w io.Writer, firstRun bool) (prefix string, err error) {
	c.Tracef("cogging plaintext")
	mark := c.Opt.StartMark + "gocog"
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
	c.Tracef("Wrote %c lines to output file", len(lines))

	if !found {
		return "", err
	}

	return getPrefix(lines, mark), err
}

// Reads lines from the reader until reaching the gocog endmark
// Writes out the generator code to a file with the given name
// any lines that start with whitespace and then prefix will have
// prefix replaced by an equal number of spaces
func (c *context) cogGeneratorCode(r *bufio.Reader, w io.Writer, prefix string) error {
	c.Tracef("cogging generator code")
	lines, _, err := readUntil(r, "gocog"+c.Opt.EndMark)
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
	c.Tracef("Wrote %c lines to output file", len(lines))

	if !c.Opt.Excise && len(lines) > 0 {
		if err := c.generate(w, lines[:len(lines)-1], prefix); err != nil {
			return err
		}
	}

	return nil
}

func (c *context) generate(w io.Writer, lines []string, prefix string) error {
	gen := fmt.Sprintf("%s_cog_%s", c.File, c.Opt.Ext)
	defer os.Remove(gen)

	// write all but the last line to the generator file
	if err := writeNewFile(gen, lines, prefix); err != nil {
		return err
	}

	b := bytes.Buffer{}
	if err := c.runFile(gen, &b); err != nil {
		return err
	}
	if _, err := w.Write(b.Bytes()); err != nil {
		return err
	}

	// make sure we always end with a newline so we keep [[[end]]] on its own line
	if b.Len() > 0 && b.Bytes()[b.Len()-1] != newline {
		if _, err := w.Write([]byte{newline}); err != nil {
			return err
		}
	}
	return nil
}

func (c *context) runFile(f string, w io.Writer) error {
	cmd := c.Opt.Command
	if strings.Contains(cmd, "%s") {
		cmd = fmt.Sprintf(cmd, f)
	}
	args := make([]string, len(c.Opt.Args), len(c.Opt.Args))
	for i, s := range c.Opt.Args {
		if strings.Contains(s, "%s") {
			args[i] = fmt.Sprintf(s, f)
		} else {
			args[i] = s
		}
	}

	if err := run(cmd, args, w, c.log); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

func (c *context) cogToEnd(r *bufio.Reader, w io.Writer) error {
	c.Tracef("cogging to end")
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, found, err := findLine(r, c.Opt.StartMark+"end"+c.Opt.EndMark)
	if err == io.EOF && !found {
		if !c.Opt.UseEOF {
			return UnexpectedEOF
		}
		c.Tracef("No gocog end statement, treating EOF as end statement.")
		return io.EOF
	}
	if err != nil && err != io.EOF {
		return err
	}

	// if there's no error, found should always be true, so just write out
	if _, err := w.Write([]byte(line)); err != nil {
		return err
	}
	c.Tracef("Wrote 1 line to output file")
	return err
}
