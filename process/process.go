package process

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	START = iota
	END
	COG_END
)

var markers = []string{"[[[cog", "]]]", "[[[end]]]"}

func Filename(f string, opt *Options) error {
	in, err := os.Open(f)
	if err != nil {
		log.Println("Error reading file", f, err)
		return err
	}
	defer in.Close()
	r := bufio.NewReader(in)

	log.Println("Processing file", f)
	return Cog(r, f, opt)
}

func Cog(r *bufio.Reader, f string, opt *Options) error {
	tmpFile := f + "_cog"
	out, err := createNew(tmpFile)
	if err != nil {
		return fmt.Errorf("Error creating cog output file: %s", err)
	}

	w := bufio.NewWriter(out)

	output, err := cog(r, w, f, opt)
	out.Close()

	if err == io.EOF {
		if output {
			if err := os.Remove(f); err != nil {
				return fmt.Errorf("Error removing original file %s: %s", f, err)
			}
			if err := os.Rename(tmpFile, f); err != nil {
				return fmt.Errorf("Error renaming cog file %s: %s", tmpFile, err)
			}
		}
		if err := os.Remove(tmpFile); err != nil {
			return fmt.Errorf("Error removing empty cog file %s: %s", tmpFile, err)
		}
	} else {
		if err := os.Remove(tmpFile); err != nil {
			log.Printf("Error removing partial cog file %s: %s", tmpFile, err)
		}
		return err
	}
	return nil
}

func cog(r *bufio.Reader, w *bufio.Writer, name string, opt *Options) (bool, error) {
	currentLine := 0
	for {
		if err := cogPlainText(r, w, name, &currentLine); err != nil {
			return currentLine != 0, err
		}

		if err := cogGeneratorCode(r, w, name, &currentLine); err != nil {
			return true, err
		}

		if err := cogToEnd(r, w, name, &currentLine, opt.UseEOF); err != nil {
			return true, err
		}
	}
	panic("Can't get here!")
}

func cogToEnd(r *bufio.Reader, w *bufio.Writer, name string, currentLine *int, useEOF bool) error {
	// we'll drop all but the COG_END line, so no need to keep them in memory
	line, count, err := findLine(r, COG_END, *currentLine)
	*currentLine += count
	if err == io.EOF && !useEOF {
		return fmt.Errorf("Unexpected EOF while looking for %s in file %s", markers[COG_END], name)
	}
	if err != nil && err != io.EOF {
		return err
	}

	_, err = w.Write([]byte(line))
	if err != nil {
		return fmt.Errorf("Error writing to output file for %s: %s", name, err)
	}
}

func cogPlainText(r *bufio.Reader, w *bufio.Writer, name string, currentLine *int) error {
	lines, err := readUntil(r, START, *currentLine)
	if err != nil && err != io.EOF {
		return err
	}

	if err == io.EOF && *currentLine == 0 {
		// default case - no cog code, don't bother to write out anything
		return nil
	}

	*currentLine += len(lines)

	// we can just write out the non-cog code to the output file
	// this also writes out the cog start line
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return fmt.Errorf("Error writing to output file for %s: %s", name, err)
		}
	}

	return err
}

func cogGeneratorCode(r *bufio.Reader, w *bufio.Writer, name string, currentLine *int) error {
	startLine := *currentLine

	lines, err := readUntil(r, END, startLine)
	if err == io.EOF {
		return fmt.Errorf("Unexpected EOF while looking for %s in file %s", markers[END], name)
	}
	if err != nil {
		return err
	}

	*currentLine += len(lines)

	// we have to write this out both to the output file and to the code file that we'll be running
	for _, line := range lines {
		if _, err := w.Write([]byte(line)); err != nil {
			return fmt.Errorf("Error writing to output file for %s: %s", name, err)
		}
	}

	// todo: handle other languages
	gen := fmt.Sprintf("%s_cog_%d%s", name, startLine, ".go")

	// write all but the last line to the generator file
	if err := writeAllNew(gen, lines[:len(lines)-1]); err != nil {
		return fmt.Errorf("Error writing code generation file: %s", err)
	}

	if err := generate(w, gen); err != nil {
		return fmt.Errorf("Error generating code from source: %s", err)
	}
	return nil
}

func generate(w *bufio.Writer, name string) error {
	return nil
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

func readUntil(r *bufio.Reader, marker int, startLine int) ([]string, error) {
	lines := make([]string, 100)
	var err error
	for err == nil {
		line, err := r.ReadString('\n')
		lines = append(lines, line)
		if err == nil || err == io.EOF {
			for i, s := range markers {
				if strings.Contains(line, s) {
					if i == marker {
						return lines, nil
					}
					return lines, fmt.Errorf("Unexpected Cog marker %s on line %d", markers[i], startLine+len(lines))
				}
			}
		}
	}
	return lines, err
}

func findLine(r *bufio.Reader, marker int, startLine int) (string, int, error) {
	count := 0
	var err error
	for err == nil {
		line, err := r.ReadString('\n')
		count++
		if err == nil || err == io.EOF {
			for i, s := range markers {
				if strings.Contains(line, s) {
					if i == marker {
						return line, count, nil
					}
					return line, count, fmt.Errorf("Unexpected Cog marker %s on line %d", markers[i], startLine+count)
				}
			}
		}
	}
	return line, count, err
}

func createNew(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return nil, fmt.Errorf("File '%s' already exists.", filename)
	}
	return f, nil
}
