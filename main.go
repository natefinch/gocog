/* {{{gocog
package main
import(
  "fmt"
  "os"
  "os/exec"
)
func main() {
  fmt.Println("")
  fmt.Print("/", "*", "\n")
  fmt.Println("Package main creates an executable that will generate text from inline sourcecode.\n")
  cmd := exec.Command("gocog")
  cmd.Stdout = os.Stdout
  cmd.Run()
  fmt.Print("*","/", "\n")
  fmt.Println("package main")
}
gocog}}} */

/*
Package main creates an executable that will generate text from inline sourcecode.

Usage:
  gocog [OPTIONS] [INFILE1 | @FILELIST1] ...

  Runs gocog over each infile. 
  Filenames prepended with @ are assumed to be newline delimited lists of files to be processed.

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
*/
package main
