package main

import (
	"os"
	"os/exec"
	"testing"
)

// This helper will be run in a subprocess to test the exit builtin.
func TestHelperExit(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Call the builtin exit command.
	builtins["exit"]([]string{"exit"}, os.Stdout, os.Stderr, os.Stdin)
}

func TestBuiltinExit(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperExit")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	err := cmd.Run()

	// os.Exit(0) returns an error of type *exec.ExitError with ExitCode 0
	if err == nil {
		// Process exited with code 0, as expected
		return
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 0 {
			t.Fatalf("exit code = %d, want 0", exitErr.ExitCode())
		}
	} else {
		t.Fatalf("unexpected error: %v", err)
	}
}
