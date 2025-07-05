package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/chzyer/readline"
)

// DummyCompleter implements PrefixCompleterInterface for testing
type DummyCompleter struct {
	suggestions [][]rune
	length      int
}

func (d *DummyCompleter) Do(line []rune, pos int) ([][]rune, int) {
	return d.suggestions, d.length
}

func (d *DummyCompleter) GetChildren() []readline.PrefixCompleterInterface {
	return nil
}

func (d *DummyCompleter) GetName() []rune {
	return []rune{}
}

func (d *DummyCompleter) Print(prefix string, level int, buf *bytes.Buffer) {
	// No-op for dummy
}

func (d *DummyCompleter) SetChildren(children []readline.PrefixCompleterInterface) {
	// No-op for dummy
}

func TestBellCompleter_NoSuggestions_RingsBellAndResetsTabCount(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	bc := &bellCompleter{
		PrefixCompleterInterface: &DummyCompleter{suggestions: nil, length: 0},
	}
	bc.tabCount = 42 // set to nonzero to check reset
	bc.lastLine = "old"
	bc.Do([]rune("foo"), 3)

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)

	if !strings.Contains(buf.String(), "\a") {
		t.Errorf("expected bell character to be printed")
	}
	if bc.tabCount != 0 {
		t.Errorf("tabCount = %d, want 0", bc.tabCount)
	}
	if bc.lastLine != "foo" {
		t.Errorf("lastLine = %q, want %q", bc.lastLine, "foo")
	}
}
