package processor

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"testing"
)

type CPTData struct {
	input  string
	output string
	prefix string
	first  bool
	err    error
}

func TestCogPlainText(t *testing.T) {
	c := &context{"foo", &Options{StartMark: "[[["}, log.New(ioutil.Discard, "", log.LstdFlags)}

	tests := []CPTData{
		{"", "", "", true, NoCogCode},
		{"a\nb\nc", "", "", true, NoCogCode},
		{"a\nb\n[[[gocog", "", "", true, UnexpectedEOF},
		{"a\nb\n[[[gocog\n", "a\nb\n[[[gocog\n", "", true, nil},
		{"a\nb\n[[[gocog  stuff\n and more stuff\n", "a\nb\n[[[gocog  stuff\n", "", true, nil},
		{"a\nb\n// [[[gocog\n", "a\nb\n// [[[gocog\n", "// ", true, nil},
	}

	for i, test := range tests {
		_ = i
		in := bytes.NewBufferString(test.input)
		out := &bytes.Buffer{}

		r := bufio.NewReader(in)
		prefix, err := c.cogPlainText(r, out, test.first)

		if prefix != test.prefix {
			t.Errorf("CogPlainText Test %d: Expected prefix: '%s', Got prefix: '%s'", i, test.prefix, prefix)
		}

		if err != test.err {
			t.Errorf("CogPlainText Test %d: Expected error: '%v', Got error: '%v'", i, test.err, err)
		}

		output := out.String()
		if output != test.output {
			t.Errorf("CogPlainText Test %d: Expected output:\n'%s'\nGot output:\n'%s'", i, test.output, output)
		}
	}
}

type CTEData struct {
	input  string
	output string
	useEOF bool
	err    error
}

func TestCogToEnd(t *testing.T) {
	tests := []CTEData{
		{"", "", false, UnexpectedEOF},
		{"", "", true, io.EOF},
		{"1\n2\n[[[end]]]", "[[[end]]]", false, io.EOF},
		{"1\n2\n[[[end]]]\n", "[[[end]]]\n", false, nil},
		{"1\n2", "", true, io.EOF},
		{"1\n2", "", false, UnexpectedEOF},
		{"1\n2\n// [[[end]]]\n", "// [[[end]]]\n", false, nil},
	}

	opts := &Options{
		StartMark: "[[[",
		EndMark:   "]]]",
	}
	c := &context{"foo", opts, log.New(ioutil.Discard, "", log.LstdFlags)}

	for i, test := range tests {
		opts.UseEOF = test.useEOF

		in := bytes.NewBufferString(test.input)
		out := &bytes.Buffer{}

		r := bufio.NewReader(in)
		err := c.cogToEnd(r, out)

		if err != test.err {
			t.Errorf("CogToEnd Test %d: Expected error %v, got %v", i, test.err, err)
		}

		output := out.String()
		if output != test.output {
			t.Errorf("CogToEnd Test %d: Expected output:\n'%s'\nGot output:\n'%s'", i, test.output, output)
		}

	}

}
