// cog - generate code with inlined Go code.
package main

import (
	"cog/process"
	flags "github.com/jessevdk/go-flags"
	"os"
	"runtime"
)

var infile struct {
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	opts := new(process.Options)
	p := flags.NewParser(opts, flags.Default)
	p.Usage = "[OPTIONS] [INFILE | @FILELIST] ..."

	remaining, err := p.ParseArgs(os.Args)
	if err != nil {
		os.Exit(1)
	}
	// strip off the executable name
	remaining = remaining[1:]
	for _, s := range remaining {
		if opts.Serial {
			process.Filename(s, opts)
		} else {
			go process.Filename(s, opts)
		}
	}
}
