package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuiltinType_Builtin(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: type echo
	args := []string{"type", "echo"}
	err := builtins["type"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("type returned error: %v", err)
	}

	got := out.String()
	want := "echo is a shell builtin\n"
	if got != want {
		t.Errorf("type output = %q, want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("type wrote to stderr: %q", errOut.String())
	}
}

func TestBuiltinType_External(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: type ls (assuming ls exists in PATH)
	args := []string{"type", "ls"}
	err := builtins["type"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("type returned error: %v", err)
	}

	got := out.String()
	if !strings.HasPrefix(got, "ls is ") {
		t.Errorf("type output = %q, want prefix %q", got, "ls is ")
	}
	if errOut.Len() != 0 {
		t.Errorf("type wrote to stderr: %q", errOut.String())
	}
}

func TestBuiltinType_NotFound(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: type notarealcommand
	args := []string{"type", "notarealcommand"}
	err := builtins["type"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("type returned error: %v", err)
	}

	got := out.String()
	want := "notarealcommand: not found\n"
	if got != want {
		t.Errorf("type output = %q, want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("type wrote to stderr: %q", errOut.String())
	}
}

func TestBuiltinType_TooManyArgs(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	// Simulate: type a b
	args := []string{"type", "a", "b"}
	err := builtins["type"](args, &out, &errOut, nil)
	if err == nil {
		t.Errorf("type should return error for too many arguments")
	}
	got := errOut.String()
	want := "type: too many arguments\n"
	if got != want {
		t.Errorf("type stderr = %q, want %q", got, want)
	}
}
