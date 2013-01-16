// cog - generate code with inlined Go code.
package main

import (
	"github.com/NateFinch/gocog/process"
	flags "github.com/jessevdk/go-flags"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

var infile struct {
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	opts := new(process.Options)
	p := flags.NewParser(opts, flags.Default)
	p.Usage = `[OPTIONS] [INFILE1] [@FILELIST1] ...

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
			if names, err := readFile(s[1:]); err == nil {
				files = append(files, names...)
			}
		} else {
			files = append(files, s)
		}
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(files))
	for _, s := range files {
		if opts.Serial {
			process.Cog(s, opts, wg)
		} else {
			go process.Cog(s, opts, wg)
		}
	}
	wg.Wait()
}

func readFile(name string) ([]string, error) {
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
