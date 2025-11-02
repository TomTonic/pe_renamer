package main

import (
	"strings"
	"testing"
)

func Test_VersionOutput(t *testing.T) {
	// backup
	oldTag := gitTag
	oldOS := buildOS
	oldArch := buildArch
	defer func() {
		gitTag = oldTag
		buildOS = oldOS
		buildArch = oldArch
	}()

	gitTag = "v9.9.9-test"
	buildOS = "linux"
	buildArch = "amd64"

	var sb strings.Builder
	PrintVersion(&sb)
	out := sb.String()

	if !strings.Contains(out, "OS: linux") {
		t.Fatalf("expected OS in version output, got: %s", out)
	}
	if !strings.Contains(out, "ARCH: amd64") {
		t.Fatalf("expected ARCH in version output, got: %s", out)
	}
	if !strings.Contains(out, "TAG: v9.9.9-test") {
		t.Fatalf("expected TAG in version output, got: %s", out)
	}
}
