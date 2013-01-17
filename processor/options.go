package processor

type Options struct {
	UseEOF    bool     `short:"z" long:"eof" description:"The end marker can be assumed at eof."`
	Verbose   bool     `short:"v" long:"verbose" description:"enables verbose output"`
	Quiet     bool     `short:"q" long:"quiet" description:"turns off all output"`
	Serial    bool     `short:"S" long:"serial" description:"Write to the specified cog files serially"`
	Command   string   `short:"c" long:"cmd" description:"The command used to run the generator code"`
	Args      []string `short:"a" long:"args" description:"Comma separated arguments to cmd, %s for the code file"`
	Ext       string   `short:"e" long:"ext" description:"Extension to append to the generator filename"`
	StartMark string   `short:"M" long:"startmark" description:"String that starts gocog statements"`
	EndMark   string   `short:"E" long:"endmark" description:"String that ends gocog statements"`
	//	Checksum bool              `short:"c" description:"Checksum the output to protect it against accidental change."`
	//	Delete   bool              `short:"d" description:"Delete the generator code from the output file."`
	//	Define   map[string]string `short:"D" description:"Define a global string available to your generator code."`
	//	Include  string            `short:"I" description:"Add PATH to the list of directories for data files and modules."`
	//	Output   string            `short:"o" description:"Write the output to OUTNAME."`
	//	Suffix   string            `short:"s" description:"Suffix all generated output lines with STRING."`
	//	Unix     bool              `short:"U" description:"Write the output with Unix newlines (only LF line-endings)."`
	//	WriteCmd string            `short:"w" description:"Use CMD if the output file needs to be made writable. A %s in the CMD will be filled with the filename."`
	//  Excise bool `short:"x" description:"Excise all the generated output without running the generators."`
}
