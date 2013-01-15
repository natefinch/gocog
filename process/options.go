package process

type Options struct {
	//	Checksum bool              `short:"c" description:"Checksum the output to protect it against accidental change."`
	//	Delete   bool              `short:"d" description:"Delete the generator code from the output file."`
	//	Define   map[string]string `short:"D" description:"Define a global string available to your generator code."`
	//	Empty    bool              `short:"e" description:"Warn if a file has no cog code in it."`
	//	Include  string            `short:"I" description:"Add PATH to the list of directories for data files and modules."`
	//	Output   string            `short:"o" description:"Write the output to OUTNAME."`
	//	Replace  bool              `short:"r" description:"Replace the input file with the output."`
	//	Suffix   string            `short:"s" description:"Suffix all generated output lines with STRING."`
	//	Unix     bool              `short:"U" description:"Write the output with Unix newlines (only LF line-endings)."`
	//	WriteCmd string            `short:"w" description:"Use CMD if the output file needs to be made writable. A %s in the CMD will be filled with the filename."`
	// Excise bool `short:"x" description:"Excise all the generated output without running the generators."`
	UseEOF bool `short:"z" description:"The [[[end]]] marker can be omitted, and is assumed at eof."`
	//	Version  bool              `short:"v" description:"Print the version of cog and exit"`
	// Serial bool `short:"S" description:"Write to the specified cog files serially (default is parallel)"`
}
