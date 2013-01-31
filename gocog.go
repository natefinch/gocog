package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/kballard/go-shellquote"
	"github.com/natefinch/gocog/processor"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

const (
	version = "gocog v1.0 build %s\n"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	opts := &processor.Options{
		Command:   "go",
		Args:      []string{"run", "%s"},
		Ext:       ".go",
		StartMark: "[[[",
		EndMark:   "]]]",
	}

	p := flags.NewParser(opts, flags.Default)
	p.Usage = `[OPTIONS] [INFILE | @FILELIST] ...

  Runs gocog over each infile. 
  Strings prepended with @ are assumed to be files continaing newline delimited lists of gocog command lines.`

	remaining, err := p.ParseArgs(os.Args[1:])
	if err != nil {
		log.Println("Error parsing args:", err)
		os.Exit(1)
	}

	ver := ""
	// {{{gocog
	// package main
	// import (
	//   "fmt"
	//   "time"
	// )
	// func main() {
	// 	t := time.Now()
	// 	fmt.Printf("\tver = \"%d%02d%02d\"\n", t.Year(), int(t.Month()), t.Day())
	// }
	// gocog}}}
	ver = "20130125"
	// {{{end}}}
	if opts.Version {
		fmt.Printf(version, ver)
		os.Exit(0)
	}

	if len(remaining) < 1 {
		p.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	procs, err := handleCommandLine(os.Args[1:], opts)
	if err != nil {
		p.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(procs))
	for _, p := range procs {
		if opts.Serial {
			run(p, wg)
		} else {
			go run(p, wg)
		}
	}
	wg.Wait()
}

func run(p *processor.Processor, wg *sync.WaitGroup) {
	p.Run()
	wg.Done()
}

func handleCommandLine(args []string, opts *processor.Options) ([]*processor.Processor, error) {
	p := flags.NewParser(opts, flags.Default)

	remaining, err := p.ParseArgs(args)
	if err != nil {
		return nil, err
	}

	if len(remaining) < 1 {
		return nil, errors.New("No files targeted on command line")
	}

	if len(opts.Ext) > 0 && opts.Ext[:1] != "." {
		opts.Ext = "." + opts.Ext
	}

	return handleRemaining(remaining, opts)
}

func handleRemaining(names []string, opts *processor.Options) ([]*processor.Processor, error) {
	procs := make([]*processor.Processor, 0, len(names))
	for _, s := range names {
		if s[:1] == "@" {
			name := s[1:]
			p, err := handleFilelist(name, opts)
			if err != nil {
				return nil, err
			}
			procs = append(procs, p...)
		} else {
			procs = append(procs, processor.New(s, opts))
		}
	}
	return procs, nil
}

func handleFilelist(name string, opts *processor.Options) ([]*processor.Processor, error) {
	if opts.Verbose {
		log.Printf("Processing filelist '%s'", name)
	}
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	lines := strings.SplitAfter(string(b), "\n")

	procs := make([]*processor.Processor, 0, len(lines))
	for i, line := range lines {
		args, err := shellquote.Split(line)
		if err != nil {
			return nil, fmt.Errorf("Error parsing command line in filelist '%s' line %d", name, i+1)
		}
		p, err := handleCommandLine(args, opts)
		if err != nil {
			return nil, err
		}
		procs = append(procs, p...)
	}
	return procs, nil
}
