package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestBuiltinPwd(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: pwd
	args := []string{"pwd"}
	err := builtins["pwd"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("pwd returned error: %v", err)
	}

	got := strings.TrimSpace(out.String())
	want, _ := os.Getwd()
	if got != want {
		t.Errorf("pwd output = %q, want %q", got, want)
	}

	if errOut.Len() != 0 {
		t.Errorf("pwd wrote to stderr: %q", errOut.String())
	}
}
