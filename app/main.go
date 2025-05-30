package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

// findExecutable searches for an executable in the PATH and returns its full path if found, or an empty string if not found.
func findExecutable(cmd string) string {
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	for _, dir := range paths {
		fullPath := dir + string(os.PathSeparator) + cmd
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

// Helper to split command line with single quote support (concatenates adjacent quoted args)
func parseMetas(input string) []string {

	var args []string
	var buf strings.Builder
	inSingleQuotes, inDoubleQuotes := false, false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			} else {
				buf.WriteByte(ch)
			}
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			} else {
				buf.WriteByte(ch)
			}
		case '\\':
			if inDoubleQuotes && i+1 < len(input) {
				next := input[i+1]
				// Only escape \, $, " or newline inside double quotes
				if next == '\\' || next == '$' || next == '"' || next == '\n' {
					i++
					buf.WriteByte(next)
				} else {
					buf.WriteByte(ch)
				}
			} else if !inSingleQuotes && !inDoubleQuotes && i+1 < len(input) {
				i++
				buf.WriteByte(input[i])
			} else {
				buf.WriteByte(ch)
			}
		case ' ':
			if inSingleQuotes || inDoubleQuotes {
				buf.WriteByte(ch)
			} else if buf.Len() > 0 {
				args = append(args, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteByte(ch)
		}

	}

	if buf.Len() > 0 {
		args = append(args, buf.String())
	}

	return args
}

var builtins = make(map[string]func([]string, io.Writer, io.Writer) error)

func init() {
	builtins["exit"] = func(args []string, stdout, stderr io.Writer) error {
		os.Exit(0)
		return nil
	}
	builtins["pwd"] = func(args []string, stdout, stderr io.Writer) error {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "pwd: %v\n", err)
			return err
		}
		fmt.Fprintln(stdout, cwd)
		return nil
	}
	builtins["cd"] = func(args []string, stdout, stderr io.Writer) error {
		if len(args) != 2 {
			fmt.Fprintln(stderr, "cd: too many arguments")
			return fmt.Errorf("cd: too many arguments")
		}
		arg := args[1]
		if arg == "~" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintln(stderr, "cd: cannot determine home directory")
				return err
			}
			arg = home
		}
		absPath, err := os.Stat(arg)
		if err != nil || !absPath.IsDir() {
			fmt.Fprintf(stderr, "cd: %s: No such file or directory\n", arg)
			return fmt.Errorf("cd: %s: No such file or directory", arg)
		}
		if err := os.Chdir(arg); err != nil {
			fmt.Fprintf(stderr, "cd: %s: No such file or directory\n", arg)
			return err
		}
		return nil
	}
	builtins["echo"] = func(args []string, stdout, stderr io.Writer) error {
		fmt.Fprintln(stdout, strings.Join(args[1:], " "))
		return nil
	}
	builtins["type"] = func(args []string, stdout, stderr io.Writer) error {
		if len(args) != 2 {
			fmt.Fprintln(stderr, "type: too many arguments")
			return fmt.Errorf("type: too many arguments")
		}
		arg := args[1]
		if _, ok := builtins[arg]; ok {
			fmt.Fprintf(stdout, "%s is a shell builtin\n", arg)
		} else {
			fullPath := findExecutable(arg)
			if fullPath != "" {
				fmt.Fprintf(stdout, "%s is %s\n", arg, fullPath)
			} else {
				fmt.Fprintf(stdout, "%s: not found\n", arg)
			}
		}
		return nil
	}
}

type bellCompleter struct {
	readline.PrefixCompleterInterface
	lastLine string
	tabCount int
}

func (b *bellCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// Use the default completer
	suggestions, length := b.PrefixCompleterInterface.Do(line, pos)
	input := string(line[:pos])

	// No suggestions: ring bell as before
	if len(suggestions) == 0 {
		fmt.Print("\a")
		b.tabCount = 0
		b.lastLine = input
		return suggestions, length
	}

	// Track repeated tab presses for the same input
	if input == b.lastLine {
		b.tabCount++
	} else {
		b.tabCount = 1
		b.lastLine = input
	}

	// If multiple suggestions, handle completion to longest common prefix
	if len(suggestions) > 1 {
		// Find the longest common prefix among all suggestions
		lcp := string(suggestions[0])
		for _, s := range suggestions[1:] {
			lcp = commonPrefix(lcp, string(s))
		}
		// If the longest common prefix is longer than the current input, complete to it
		if lcp != "" && lcp != input {
			// Complete to the longest common prefix
			return [][]rune{[]rune(lcp)}, len(input)
		}
		// Otherwise, handle double-tab as before
		if b.tabCount == 1 {
			fmt.Print("\a")
			return nil, 0
		} else if b.tabCount == 2 {
			fmt.Println()
			// Collect all full suggestions
			var names []string
			for _, s := range suggestions {
				names = append(names, input+string(s))
			}
			// Sort lexicographically
			sort.Strings(names)
			for i, name := range names {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(name)
			}
			fmt.Println()
			fmt.Printf("$ %s", input)
			return nil, 0
		}
	}

	// Default: use prefix completer (for builtins, single match, etc)
	return suggestions, length
}

// Helper to get all executable names in $PATH
func getExternalCommands() []string {
	cmds := make(map[string]struct{})
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	for _, dir := range paths {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := file.Name()
			fullPath := dir + string(os.PathSeparator) + name
			info, err := os.Stat(fullPath)
			if err == nil && info.Mode().IsRegular() && (info.Mode().Perm()&0111 != 0) {
				cmds[name] = struct{}{}
			}
		}
	}
	// Convert map to slice
	var result []string
	for name := range cmds {
		result = append(result, name)
	}
	return result
}

type ShellCmd struct {
	RunFunc func() error
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

func (c *ShellCmd) Run() error {
	origStdin, origStdout, origStderr := os.Stdin, os.Stdout, os.Stderr
	if c.Stdin != nil {
		if f, ok := c.Stdin.(*os.File); ok {
			os.Stdin = f
		}
	}
	if c.Stdout != nil {
		if f, ok := c.Stdout.(*os.File); ok {
			os.Stdout = f
		}
	}
	if c.Stderr != nil {
		if f, ok := c.Stderr.(*os.File); ok {
			os.Stderr = f
		}
	}
	err := c.RunFunc()
	os.Stdin, os.Stdout, os.Stderr = origStdin, origStdout, origStderr
	return err
}

func makeShellCmd(tokens []string) *ShellCmd {
	if handler, ok := builtins[tokens[0]]; ok {
		args := tokens
		return &ShellCmd{
			RunFunc: func() error { return handler(args, os.Stdout, os.Stderr) },
			Stdin:   os.Stdin,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		}
	}
	exe := findExecutable(tokens[0])
	if exe == "" {
		return nil
	}
	cmd := exec.Command(exe, tokens[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return &ShellCmd{
		RunFunc: cmd.Run,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

func main() {
	// Prepare a list of builtin names for completion
	builtinNames := []string{}
	for name := range builtins {
		builtinNames = append(builtinNames, name)
	}

	// Gather external commands
	externalCmds := getExternalCommands()

	// Combine builtins and external commands, removing duplicates
	cmdSet := make(map[string]struct{})
	for _, name := range builtinNames {
		cmdSet[name] = struct{}{}
	}
	for _, name := range externalCmds {
		cmdSet[name] = struct{}{}
	}
	var allCommands []string
	for name := range cmdSet {
		allCommands = append(allCommands, name)
	}

	// Build the prefix completer with all commands
	prefixCompleter := readline.NewPrefixCompleter(
		func() []readline.PrefixCompleterInterface {
			var pcs []readline.PrefixCompleterInterface
			for _, name := range allCommands {
				pcs = append(pcs, readline.PcItem(name))
			}
			return pcs
		}()...,
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt: "$ ",
		AutoComplete: &bellCompleter{
			PrefixCompleterInterface: prefixCompleter,
			lastLine:                 "",
			tabCount:                 0,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize readline:", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			break
		}
		command := strings.TrimRight(line, "\r\n")

		tokens := parseMetas(command)
		if len(tokens) == 0 {
			continue
		}
		// Detect pipeline
		pipeIdx := -1
		for i, t := range tokens {
			if t == "|" {
				pipeIdx = i
				break
			}
		}

		redirectIdx, errorRedirectIdx := -1, -1
		redirectOp, errorRedirectOp := "", ""
		for i, t := range tokens {
			if t == ">" || t == "1>" || t == ">>" || t == "1>>" {
				redirectIdx = i
				redirectOp = t
				break
			} else if t == "2>" || t == "2>>" {
				errorRedirectIdx = i
				errorRedirectOp = t
				break
			}
		}

		var outBuf, errBuf bytes.Buffer

		if pipeIdx != -1 && pipeIdx+1 < len(tokens) {
			leftTokens := tokens[:pipeIdx]
			rightTokens := tokens[pipeIdx+1:]

			pr, pw := io.Pipe()
			leftCmd := makeShellCmd(leftTokens)
			rightCmd := makeShellCmd(rightTokens)
			if leftCmd == nil || rightCmd == nil {
				fmt.Fprintln(os.Stderr, "command not found")
				continue
			}
			leftCmd.Stdout = pw
			rightCmd.Stdin = pr

			done := make(chan struct{})
			go func() {
				leftCmd.Run()
				pw.Close()
				close(done)
			}()
			rightCmd.Run()
			pr.Close()
			<-done
			continue
		}

		cmd := makeShellCmd(tokens)
		if cmd == nil {
			fmt.Fprintf(os.Stderr, "%s: command not found\n", tokens[0])
			continue
		}

		// Set up output buffers for redirection
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		cmd.Stdin = os.Stdin
		cmd.Run()

		// Handle redirection (only the first redirect operator found)
		if redirectIdx != -1 && redirectIdx+1 < len(tokens) {
			var f *os.File
			var err error
			if redirectOp == ">>" || redirectOp == "1>>" {
				f, err = os.OpenFile(tokens[redirectIdx+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			} else {
				f, err = os.Create(tokens[redirectIdx+1])
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[redirectIdx+1], err)
			} else {
				f.Write(outBuf.Bytes())
				f.Close()
			}
			os.Stderr.Write(errBuf.Bytes())
		} else if errorRedirectIdx != -1 && errorRedirectIdx+1 < len(tokens) {
			var f *os.File
			var err error
			if errorRedirectOp == "2>>" {
				f, err = os.OpenFile(tokens[errorRedirectIdx+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			} else {
				f, err = os.Create(tokens[errorRedirectIdx+1])
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[errorRedirectIdx+1], err)
			} else {
				f.Write(errBuf.Bytes())
				f.Close()
			}
			os.Stdout.Write(outBuf.Bytes())
		} else {
			os.Stdout.Write(outBuf.Bytes())
			os.Stderr.Write(errBuf.Bytes())
		}
	}
}

func commonPrefix(s1, s2 string) string {
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}
	for i := 0; i < minLen; i++ {
		if s1[i] != s2[i] {
			return s1[:i]
		}
	}
	return s1[:minLen]
}
