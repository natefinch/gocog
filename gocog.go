/* Package main creates an executable that will generate text from inline sourcecode.

Usage:
  gocog [OPTIONS] [INFILE1] [@FILELIST] ...

  Runs gocog over each infile.
  Filenames prepended with @ are assumed to be newline delimited lists of files to be processed.

Help Options:
  -h, --help    Show this help message

Application Options:
  -z        The [[[end]]] marker can be omitted, and is assumed at eof.
  -v        toggles verbose output (overridden by -q)
  -q        turns off all output
  -S        Write to the specified cog files serially (default is parallel)
*/
package main

import (
	flags "github.com/jessevdk/go-flags"
	"github.com/natefinch/gocog/processor"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	opts := &processor.Options{}
	p := flags.NewParser(opts, flags.Default)
	p.Usage = `[OPTIONS] [INFILE1 | @FILELIST1] ...

  Runs gocog over each infile. 
  Filenames prepended with @ are assumed to be newline delimited lists of files to be processed.`

	remaining, err := p.ParseArgs(os.Args)
	if err != nil {
		log.Println("Error parsing args:", err)
		os.Exit(1)
	}

	// strip off the executable name
	remaining = remaining[1:]

	if len(remaining) < 1 {
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	files := make([]string, 0, len(remaining))

	for _, s := range remaining {
		if s[:1] == "@" {
			filelist := s[1:]
			if names, err := readFile(filelist, opts.Verbose); err == nil {
				if opts.Verbose {
					log.Printf("Files specified by filelist '%s': %v", filelist, names)
				}
				files = append(files, names...)
			}
		} else {
			files = append(files, s)
		}
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(files))
	if opts.Verbose {
		log.Printf("Final file processing list: %v", files)
	}
	for _, s := range files {
		if opts.Serial {
			run(s, opts, wg)
		} else {
			go run(s, opts, wg)
		}
	}
	wg.Wait()
}

func run(s string, opts *processor.Options, wg *sync.WaitGroup) {
	processor.Run(s, opts)
	wg.Done()
}

func readFile(name string, verbose bool) ([]string, error) {
	if verbose {
		log.Printf("Processing filelist '%s", name)
	}
	if b, err := ioutil.ReadFile(name); err != nil {
		log.Printf("Failed to read filelist '%s': %s", name, err)
		return []string{}, err
	} else {
		names := strings.SplitAfter(string(b), "\n")

		output := make([]string, 0, len(names))
		for _, s := range names {
			name := strings.TrimSpace(s)
			if len(name) > 0 {
				output = append(output, name)
			}
		}
		return output, nil
	}
	panic("Can't get here!")
}
