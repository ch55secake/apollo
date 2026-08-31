package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=TestVersionFlagSubprocess")
	command.Env = append(os.Environ(), "GO_WANT_VERSION_SUBPROCESS=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v\n%s", err, output)
	}
	if !strings.HasPrefix(string(output), "apollo dev\n") {
		t.Fatalf("unexpected version output: %q", output)
	}
}

func TestVersionFlagSubprocess(t *testing.T) {
	if os.Getenv("GO_WANT_VERSION_SUBPROCESS") != "1" {
		return
	}
	os.Args = []string{"apollo", "--version"}
	main()
}
