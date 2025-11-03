package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestConciseErr_Nil(t *testing.T) {
	if got := conciseErr(nil); got != "" {
		t.Fatalf("conciseErr(nil) = %q; want empty string", got)
	}
}

func TestConciseErr_NotExist(t *testing.T) {
	if got := conciseErr(os.ErrNotExist); got != "no such file or directory" {
		t.Fatalf("conciseErr(os.ErrNotExist) = %q; want %q", got, "no such file or directory")
	}
}

func TestConciseErr_PrefixStripping(t *testing.T) {
	e := errors.New("stat /some/path: permission denied")
	if got := conciseErr(e); got != "permission denied" {
		t.Fatalf("conciseErr(...) = %q; want %q", got, "permission denied")
	}
}

type errCloser struct{}

func (errCloser) Close() error { return errors.New("close failed") }

func TestMustClose_ReportsError(t *testing.T) {
	var buf bytes.Buffer
	mustClose(errCloser{}, &buf)
	s := buf.String()
	if !strings.Contains(s, "Error closing resource:") || !strings.Contains(s, "close failed") {
		t.Fatalf("mustClose output = %q; want it to contain error message", s)
	}
}
