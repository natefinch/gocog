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
	"unicode"
)

func run(cmd string, args []string, stdout io.Writer, errLog *log.Logger) error {
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

	var reg *regexp.Regexp
	if len(prefix) > 0 {
		reg = regexp.MustCompile(fmt.Sprintf(`^(\s*)%s`, regexp.QuoteMeta(prefix)))
	}

	for _, line := range lines {
		if reg != nil {
			line = reg.ReplaceAllString(line, fmt.Sprintf(`$1`))
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

func readUntil(r *bufio.Reader, marker string) (lines []string, found bool, err error) {
	lines = make([]string, 0, 50)
	for err == nil {
		var line string
		line, err = r.ReadString('\n')
		lines = append(lines, line)
		if strings.Contains(line, marker) {
			return lines, true, err
		}
	}
	return lines, false, err
}

func findLine(r *bufio.Reader, marker string) (line string, found bool, err error) {
	for err == nil {
		line, err = r.ReadString('\n')
		if err == nil || err == io.EOF {
			if strings.Contains(line, marker) {
				return line, true, err
			}
		}
	}
	return "", false, err
}

func createNew(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if os.IsExist(err) {
		return f, fmt.Errorf("File '%s' already exists.", filename)
	}
	return f, err
}

func getPrefix(lines []string, mark string) string {
	prefix := ""
	if len(lines) > 0 {
		prefix = lines[len(lines)-1]
		if i := strings.Index(prefix, mark); i > -1 {
			prefix = strings.TrimLeftFunc(prefix[:i], unicode.IsSpace)
		}
	}
	return prefix
}
