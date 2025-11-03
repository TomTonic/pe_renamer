package misc

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ConciseErr returns a short, user-friendly error message for file system errors.
// For errors produced by os.Stat it strips the leading "stat <path>: " prefix and
// returns the underlying message (for example "no such file or directory").
func ConciseErr(err error) string {
	if err == nil {
		return ""
	}
	if os.IsNotExist(err) {
		return "no such file or directory"
	}
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i != -1 {
		return s[i+2:]
	}
	return s
}

// MustClose closes the provided io.Closer and logs any error to errWriter.
// Use this in defers to make intent explicit and surface closing errors.
func MustClose(c io.Closer, errWriter io.Writer) {
	if err := c.Close(); err != nil {
		_, _ = fmt.Fprintf(errWriter, "Error closing resource: %s\n", ConciseErr(err))
	}
}
