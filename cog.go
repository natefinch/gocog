// cog - generate code with inlined Go code.
package main

import (
	"github.com/NateFinch/gocog/process"
	flags "github.com/jessevdk/go-flags"
	"log"
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
	p.Usage = "[OPTIONS] [INFILE] ..."

	remaining, err := p.ParseArgs(os.Args)
	if err != nil {
		log.Println("Error parsing args:", err)
		os.Exit(1)
	}

	// strip off the executable name
	remaining = remaining[1:]

	for _, s := range remaining {
		if err := process.Cog(s, opts); err != nil {
			log.Printf("Error processing '%s': %s", s, err)
		} else {
			log.Printf("Finished processing '%s' successfully.", s)
		}
	}
}
