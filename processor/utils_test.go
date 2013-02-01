package processor

import (
	"bufio"
	"bytes"
	"io"
	"testing"
)

type ReadUntilData struct {
	s     string
	count int
	found bool
	err   error
}

func TestReadUntil(t *testing.T) {

	tests := []ReadUntilData{
		{"a\nb\n", 3, false, io.EOF},
		{"a\nb\nEND", 3, true, io.EOF},
		{"a\nb\nEND\n", 3, true, nil},
		{"a\nb\nc\nEND\nz", 4, true, nil},
	}

	marker := "END"
	for i, test := range tests {
		_ = i

		r := bufio.NewReader(bytes.NewBufferString(test.s))

		lines, found, err := readUntil(r, marker)
		if len(lines) != test.count {
			t.Errorf("ReadUntil Test %d: Incorrect number of lines returned."+
				" Expected: %d, Got: %d", i, test.count, len(lines))
		}
		if found != test.found {
			if test.found {
				t.Errorf("ReadUntil Test %d: Failed to find existing marker.", i)
			} else {
				t.Errorf("ReadUntil Test %d: Incorrectly found non-existent marker.", i)
			}
		}
		if err != test.err {
			t.Errorf("ReadUntil Test %d: Unexpected error returned. Expected: %v, Got: %v", i, test.err, err)
		}
	}

}

type FindLineData struct {
	s     string
	line  string
	found bool
	err   error
}

func TestFindLine(t *testing.T) {

	tests := []FindLineData{
		{"a\nb\n", "", false, io.EOF},
		{"a\nb\nEND", "END", true, io.EOF},
		{"a\nb\nEND\n", "END\n", true, nil},
		{"a\nb\nc\nEND\nz", "END\n", true, nil},
	}

	marker := "END"
	for i, test := range tests {
		_ = i

		r := bufio.NewReader(bytes.NewBufferString(test.s))

		line, found, err := findLine(r, marker)
		if line != test.line {
			t.Errorf("ReadLine Test %d: Incorrect line returned. Expected: '%s', Got: '%s'", i, test.line, line)
		}
		if found != test.found {
			if test.found {
				t.Errorf("ReadLine Test %d: Failed to find existing marker.", i)
			} else {
				t.Errorf("ReadLine Test %d: Incorrectly found non-existent marker.", i)
			}
		}
		if err != test.err {
			t.Errorf("ReadLine Test %d: Unexpected error returned. Expected: %v, Got: %v", i, test.err, err)
		}
	}

}

type PrefixData struct {
	input  string
	prefix string
}

func TestGetPrefix(t *testing.T) {
	tests := []PrefixData{
		{"     // START", "// "},
		{"\t #START", "#"},
		{"START", ""},
		{"   \t  START", ""},
		{"//START", "//"},
	}

	marker := "START"
	for i, test := range tests {
		_ = i
		prefix := getPrefix(test.input, marker)
		if prefix != test.prefix {
			t.Errorf("GetPrefix Test %d: incorrect prefix returned. Expected: '%s', Got: '%s'", i, test.prefix, prefix)
		}
	}
}
