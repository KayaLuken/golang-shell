package main

import (
	"bytes"
	"testing"
)

func TestBuiltinEcho(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: echo hello world
	args := []string{"echo", "hello", "world"}
	err := builtins["echo"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("echo returned error: %v", err)
	}

	got := out.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("echo output = %q, want %q", got, want)
	}

	if errOut.Len() != 0 {
		t.Errorf("echo wrote to stderr: %q", errOut.String())
	}
}

func TestBuiltinEcho_Empty(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: echo
	args := []string{"echo"}
	err := builtins["echo"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("echo returned error: %v", err)
	}

	got := out.String()
	want := "\n"
	if got != want {
		t.Errorf("echo output = %q, want %q", got, want)
	}

	if errOut.Len() != 0 {
		t.Errorf("echo wrote to stderr: %q", errOut.String())
	}
}
