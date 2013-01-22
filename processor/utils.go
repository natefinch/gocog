package processor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"
)

func run(cmd string, args []string, stdout io.Writer, errLog *log.Logger) error {
	errLog.Printf("%v", args)
	errOut := bytes.Buffer{}
	c := exec.Command(cmd, args...)
	c.Stdout = stdout
	c.Stderr = &errOut

	err := c.Run()
	if errOut.Len() > 0 {
		errLog.Printf("%s", errOut.String())
	}
	return err
}

func writeNewFile(name string, lines []string, prefix string) error {
	out, err := createNew(name)
	if err != nil {
		return err
	}

	prefixLen := utf8.RuneCountInString(prefix)

	var reg *regexp.Regexp 
	if prefixLen > 0 {
		reg = regexp.MustCompile(fmt.Sprintf(`^(\s*)(%s)`, regexp.QuoteMeta(prefix)))
	}

	for _, line := range lines {
		if prefixLen > 0 {
			line = reg.ReplaceAllString(line, fmt.Sprintf(`$1%s`, strings.Repeat(" ", prefixLen) ))
		}
		if _, err := out.Write([]byte(line)); err != nil {
			if err2 := out.Close(); err2 != nil {
				return fmt.Errorf("Error writing to and closing newfile %s: %s%s", name, err, err2)
			}
			return fmt.Errorf("Error writing to newfile %s: %s", name, err)
		}
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("Error closing newfile %s: %s", name, err)
	}
	return nil
}

func readUntil(r *bufio.Reader, marker string) ([]string, error) {
	lines := make([]string, 0, 50)
	var err error
	for err == nil {
		var line string
		line, err = r.ReadString('\n')
		if line != "" {
			lines = append(lines, line)
		}
		if strings.Contains(line, marker) {
			return lines, err
		}
	}
	return lines, err
}

func findLine(r *bufio.Reader, marker string) (string, error) {
	var err error
	var line string
	for err == nil {
		line, err = r.ReadString('\n')
		if err == nil || err == io.EOF {
			if strings.Contains(line, marker) {
				return line, err
			}
		}
	}
	return "", err
}

func createNew(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if os.IsExist(err) {
		return f, fmt.Errorf("File '%s' already exists.", filename)
	}
	return f, err
}
