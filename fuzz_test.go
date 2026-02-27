package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	misc "pe_renamer/misc"
)

func FuzzProcessFile(f *testing.F) {
	f.Add([]byte("MZ"))
	f.Add([]byte("not-a-pe"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}

		dir := t.TempDir()
		file := filepath.Join(dir, "input.bin")
		if err := os.WriteFile(file, data, 0o600); err != nil {
			t.Fatalf("write fuzz input: %v", err)
		}

		candidates := map[string]renamingCandidate{}
		processFile(file, false, true, false, false, &candidates, &bytes.Buffer{}, &bytes.Buffer{})
	})
}

func FuzzSortCandidates(f *testing.F) {
	f.Add("a", "b", "x.exe", true, float64(10.0))
	f.Add("", "", "", false, float64(0.0))

	f.Fuzz(func(t *testing.T, path, original, newName string, extMatches bool, similarity float64) {
		if len(path) > 1024 || len(original) > 1024 || len(newName) > 1024 {
			t.Skip()
		}

		candidates := []renamingCandidate{
			{Path: path, OriginalName: original, NewName: newName, ExtMatches: extMatches, Similarity: similarity},
			{Path: "z", OriginalName: "z", NewName: "z.exe", ExtMatches: false, Similarity: 0},
			{Path: "a", OriginalName: "a", NewName: "a.dll", ExtMatches: true, Similarity: 100},
		}

		sortCandidates(candidates)
		sortCandidates(candidates)

		for i := 1; i < len(candidates); i++ {
			left := candidates[i-1]
			right := candidates[i]

			if left.ExtMatches != right.ExtMatches {
				if !left.ExtMatches && right.ExtMatches {
					t.Fatalf("invalid order by ExtMatches: %#v then %#v", left, right)
				}
				continue
			}

			if left.Similarity != right.Similarity {
				if left.Similarity < right.Similarity {
					t.Fatalf("invalid order by Similarity: %#v then %#v", left, right)
				}
				continue
			}

			if left.Path != right.Path {
				if left.Path > right.Path {
					t.Fatalf("invalid order by Path: %#v then %#v", left, right)
				}
				continue
			}

			if left.OriginalName > right.OriginalName {
				t.Fatalf("invalid order by OriginalName: %#v then %#v", left, right)
			}
		}
	})
}

func FuzzConciseErr(f *testing.F) {
	f.Add("stat some/file: no such file or directory")
	f.Add("plain error")

	f.Fuzz(func(t *testing.T, msg string) {
		if len(msg) > 4096 {
			t.Skip()
		}

		err := bytes.ErrTooLarge
		if msg != "" {
			err = &customErr{msg: msg}
		}

		_ = misc.ConciseErr(err)
		_ = misc.ConciseErr(nil)
	})
}

type customErr struct {
	msg string
}

func (e *customErr) Error() string {
	return e.msg
}
