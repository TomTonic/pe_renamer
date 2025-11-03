package misc

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestConciseErr_Nil(t *testing.T) {
	if got := ConciseErr(nil); got != "" {
		t.Fatalf("ConciseErr(nil) = %q; want empty string", got)
	}
}

func TestConciseErr_NotExist(t *testing.T) {
	if got := ConciseErr(os.ErrNotExist); got != "no such file or directory" {
		t.Fatalf("ConciseErr(os.ErrNotExist) = %q; want %q", got, "no such file or directory")
	}
}

func TestConciseErr_PrefixStripping(t *testing.T) {
	e := errors.New("stat /some/path: permission denied")
	if got := ConciseErr(e); got != "permission denied" {
		t.Fatalf("ConciseErr(...) = %q; want %q", got, "permission denied")
	}
}

type errCloser struct{}

func (errCloser) Close() error { return errors.New("close failed") }

func TestMustClose_ReportsError(t *testing.T) {
	var buf bytes.Buffer
	MustClose(errCloser{}, &buf)
	s := buf.String()
	if !strings.Contains(s, "Error closing resource:") || !strings.Contains(s, "close failed") {
		t.Fatalf("MustClose output = %q; want it to contain error message", s)
	}
}
