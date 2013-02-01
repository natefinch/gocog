// [[[gocog
// package main
// import(
//   "fmt"
//   "os"
//   "os/exec"
// )
// func main() {
//   fmt.Println("")
//   fmt.Print("/", "*", "\n")
//   fmt.Println("Command gocog creates an executable that will generate text from sourcecode inlined in another file.\n")
//   cmd := exec.Command("gocog")

//   cmd.Stdout = os.Stdout
//   cmd.Run()
//   fmt.Print("*","/", "\n")
//   fmt.Println("package documentation")
// }
// gocog]]]

/*
Command gocog creates an executable that will generate text from sourcecode inlined in another file.

Usage:
  gocog [OPTIONS] [INFILE | @FILELIST] ...

  Runs gocog over each infile. 
  Strings prepended with @ are assumed to be files continaing newline delimited lists of gocog command lines.
  Command line options are passed to each command line in the file list, but options on the file list line
  will override command line options. You may have filelists specified inside filelist files.

Help Options:
  -h, --help         Show this help message

Application Options:
  -z, --eof          The end marker can be assumed at eof.
  -v, --verbose      enables verbose output
  -q, --quiet        turns off all output
  -S, --serial       Write to the specified cog files serially
  -c, --cmd          The command used to run the generator code (go)
  -a, --args         Comma separated arguments to cmd, %s for the code file
                     ([run, %s])
  -e, --ext          Extension to append to the generator filename (.go)
  -M, --startmark    String that starts gocog statements ([[[)
  -E, --endmark      String that ends gocog statements (]]])
  -x, --excise       Excise all the generated output without running the
                     generators.
  -V, --version      Display the version of gocog
*/
package documentation
