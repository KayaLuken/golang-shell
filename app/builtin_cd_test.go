package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinCd_ValidDir(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get current dir: %v", err)
	}
	tmpDir := os.TempDir()

	args := []string{"cd", tmpDir}
	err = builtins["cd"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("cd returned error: %v", err)
	}

	// Check that we actually changed directory
	got, _ := os.Getwd()
	if got != tmpDir {
		t.Errorf("cd did not change directory: got %q, want %q", got, tmpDir)
	}

	// Restore original directory
	os.Chdir(origDir)
}

func TestBuiltinCd_InvalidDir(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	origDir, _ := os.Getwd()
	badDir := filepath.Join(os.TempDir(), "notarealdir")

	args := []string{"cd", badDir}
	err := builtins["cd"](args, &out, &errOut, nil)
	if err == nil {
		t.Errorf("cd should return error for invalid dir")
	}

	// Should not have changed directory
	got, _ := os.Getwd()
	if got != origDir {
		t.Errorf("cd changed directory on error: got %q, want %q", got, origDir)
	}
}

func TestBuiltinCd_HomeShortcut(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	origDir, _ := os.Getwd()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	args := []string{"cd", "~"}
	err = builtins["cd"](args, &out, &errOut, nil)
	if err != nil {
		t.Fatalf("cd returned error: %v", err)
	}

	got, _ := os.Getwd()
	if got != home {
		t.Errorf("cd ~ did not change to home: got %q, want %q", got, home)
	}

	// Restore original directory
	os.Chdir(origDir)
}

func TestBuiltinCd_TooManyArgs(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	args := []string{"cd", "a", "b"}
	err := builtins["cd"](args, &out, &errOut, nil)
	if err == nil {
		t.Errorf("cd should return error for too many arguments")
	}
	want := "cd: too many arguments\n"
	if errOut.String() != want {
		t.Errorf("cd stderr = %q, want %q", errOut.String(), want)
	}
}
