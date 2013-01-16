/*

Command gocog creates an executable that will generate text from sourcecode inlined in another file.

Usage:
	gocog [OPTIONS] [INFILE1 | @FILELIST] ...

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
package documentation
