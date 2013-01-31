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

	newline byte = 10
)

// Process the given file with the given options
// This will read the file, rewriting to a temporary file
// then run any embedded code, using the given options
func New(file string, opt *Options) *Processor {
	if opt == nil {
		opt = &Options{}
	}

	var logger *log.Logger
	if opt.Quiet {
		logger = log.New(ioutil.Discard, "", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	return &Processor{file, opt, logger}
}

type Processor struct {
	File string
	*Options
	*log.Logger
}

func (p *Processor) tracef(format string, v ...interface{}) {
	if p.Verbose {
		p.Printf(format, v...)
	}
}

func (p *Processor) Run() error {
	p.tracef("Processing file '%s'", p.File)

	output, err := p.tryCog()
	p.tracef("Output file: '%s'", output)

	if err == NoCogCode {
		if err := os.Remove(output); err != nil {
			p.Println(err)
		}
		p.Printf("No generator code found in file '%s'", p.File)
		return err
	}

	// this is the success case - got to the end of the file without any other errors
	if err == io.EOF {
		if err := os.Remove(p.File); err != nil {
			p.Printf("Error removing original file '%s': %s", p.File, err)
			return err
		}
		p.tracef("Renaming output file '%s' to original filename '%s'", output, p.File)
		if err := os.Rename(output, p.File); err != nil {
			p.Printf("Error renaming cog file '%s': %s", output, err)
			return err
		}
		p.Printf("Successfully processed '%s'", p.File)
		return nil
	} else {
		p.Printf("Error processing cog file '%s': %s", p.File, err)
		if output != "" {
			if err := os.Remove(output); err != nil {
				p.Println(err)
			}
		}
		return err
	}
	return nil
}

func (p *Processor) tryCog() (output string, err error) {
	in, err := os.Open(p.File)
	if err != nil {
		return "", err
	}
	defer in.Close()

	r := bufio.NewReader(in)

	output = p.File + "_cog"
	p.tracef("Writing output to %s", output)
	out, err := createNew(output)
	if err != nil {
		return "", err
	}
	defer out.Close()

	return output, p.gen(r, out)
}

func (p *Processor) gen(r *bufio.Reader, w io.Writer) error {
	firstRun := true
	for {
		prefix, err := p.cogPlainText(r, w, firstRun)
		if err != nil {
			return err
		}
		firstRun = false

		if err := p.cogGeneratorCode(r, w, prefix); err != nil {
			return err
		}

		if err := p.cogToEnd(r, w); err != nil {
			return err
		}
	}
	panic("Can't get here!")
}

func (p *Processor) cogPlainText(r *bufio.Reader, w io.Writer, firstRun bool) (prefix string, err error) {
	p.tracef("cogging plaintext")
	mark := p.StartMark + "gocog"
	lines, found, err := readUntil(r, mark)
	if err == io.EOF {
		if found {
			// found gocog statement, but nothing after it
			return "", io.ErrUnexpectedEOF
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
	p.tracef("Wrote %c lines to output file", len(lines))

	if !found {
		return "", err
	}

	return getPrefix(lines, mark), err
}

// Reads lines from the reader until reaching the gocog endmark
// Writes out the generator code to a file with the given name
// any lines that start with whitespace and then prefix will have
// prefix replaced by an equal number of spaces
func (p *Processor) cogGeneratorCode(r *bufio.Reader, w io.Writer, prefix string) error {
	p.tracef("cogging generator code")
	lines, _, err := readUntil(r, "gocog"+p.EndMark)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
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
	p.tracef("Wrote %c lines to output file", len(lines))

	if !p.Excise && len(lines) > 0 {
		if err := p.generate(w, lines[:len(lines)-1], prefix); err != nil {
			return err
		}
	}

	return nil
}

func (p *Processor) generate(w io.Writer, lines []string, prefix string) error {
	gen := fmt.Sprintf("%s_cog_%s", p.File, p.Ext)
	defer os.Remove(gen)

	// write all but the last line to the generator file
	if err := writeNewFile(gen, lines, prefix); err != nil {
		return err
	}

	b := bytes.Buffer{}
	if err := p.runFile(gen, &b); err != nil {
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

func (p *Processor) runFile(f string, w io.Writer) error {
	cmd := p.Command
	if strings.Contains(cmd, "%s") {
		cmd = fmt.Sprintf(cmd, f)
	}
	args := make([]string, len(p.Args), len(p.Args))
	for i, s := range p.Args {
		if strings.Contains(s, "%s") {
			args[i] = fmt.Sprintf(s, f)
		} else {
			args[i] = s
		}
	}

	if err := run(cmd, args, w, p.Logger); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

func (p *Processor) cogToEnd(r *bufio.Reader, w io.Writer) error {
	p.tracef("cogging to end")
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, found, err := findLine(r, p.StartMark+"end"+p.EndMark)
	if err == io.EOF && !found {
		if !p.UseEOF {
			return io.ErrUnexpectedEOF
		}
		p.tracef("No gocog end statement, treating EOF as end statement.")
		return io.EOF
	}
	if err != nil && err != io.EOF {
		return err
	}

	// if there's no error, found should always be true, so just write out
	if _, err := w.Write([]byte(line)); err != nil {
		return err
	}
	p.tracef("Wrote 1 line to output file")
	return err
}
