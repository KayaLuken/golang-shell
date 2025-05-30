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

var builtins = make(map[string]func([]string) error)

func init() {
	builtins["exit"] = func(args []string) error {
		os.Exit(0)
		return nil
	}
	builtins["pwd"] = func(args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("pwd: %v", err)
		}
		fmt.Println(cwd)
		return nil
	}
	builtins["cd"] = func(args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("cd: too many arguments")
		}
		arg := args[1]
		if arg == "~" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cd: cannot determine home directory")
			}
			arg = home
		}
		absPath, err := os.Stat(arg)
		if err != nil || !absPath.IsDir() {
			return fmt.Errorf("cd: %s: No such file or directory", arg)
		}
		if err := os.Chdir(arg); err != nil {
			return fmt.Errorf("cd: %s: No such file or directory", arg)
		}
		return nil
	}
	builtins["echo"] = func(args []string) error {
		fmt.Println(strings.Join(args[1:], " "))
		return nil
	}
	builtins["type"] = func(args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("type: too many arguments")
		}
		arg := args[1]
		if _, ok := builtins[arg]; ok {
			fmt.Printf("%s is a shell builtin\n", arg)
		} else {
			fullPath := findExecutable(arg)
			if fullPath != "" {
				fmt.Printf("%s is %s\n", arg, fullPath)
			} else {
				fmt.Printf("%s: not found\n", arg)
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
			rightTokens := tokens[pipeIdx+1:]
			rightExe := findExecutable(rightTokens[0])
			if rightExe == "" {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", rightTokens[0])
				continue
			}
			rightCmd := exec.Command(rightExe, rightTokens[1:]...)
			rightCmd.Args[0] = rightTokens[0]
			rightCmd.Stderr = os.Stderr
			rightCmd.Stdout = os.Stdout

			// If the first command was a builtin, use outBuf
			if _, ok := builtins[tokens[0]]; ok {
				rightCmd.Stdin = bytes.NewReader(outBuf.Bytes())
				rightCmd.Run()
				continue
			}

			// If the first command was external, stream output using a pipe
			leftExe := findExecutable(tokens[0])
			if leftExe == "" {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", tokens[0])
				continue
			}
			leftCmd := exec.Command(leftExe, tokens[1:pipeIdx]...)
			leftCmd.Args[0] = tokens[0]
			leftCmd.Stderr = os.Stderr

			pr, pw := io.Pipe()
			leftCmd.Stdout = pw
			rightCmd.Stdin = pr

			if err := leftCmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Left command error: %v\n", err)
				pw.Close()
				pr.Close()
				continue
			}
			if err := rightCmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Right command error: %v\n", err)
				pw.Close()
				pr.Close()
				leftCmd.Wait()
				continue
			}

			go func() {
				leftCmd.Wait()
				pw.Close()
			}()
			rightCmd.Wait()
			pr.Close()
			continue
		}

		if handler, ok := builtins[tokens[0]]; ok {
			args := tokens
			if redirectIdx != -1 {
				args = tokens[:redirectIdx]
			} else if errorRedirectIdx != -1 {
				args = tokens[:errorRedirectIdx]
			}
			func() {
				origStdout := os.Stdout
				origStderr := os.Stderr
				rStdout, wStdout, _ := os.Pipe()
				rStderr, wStderr, _ := os.Pipe()
				os.Stdout = wStdout
				os.Stderr = wStderr

				doneOut := make(chan struct{})
				doneErr := make(chan struct{})
				go func() {
					outBuf.ReadFrom(rStdout)
					close(doneOut)
				}()
				go func() {
					errBuf.ReadFrom(rStderr)
					close(doneErr)
				}()

				// FIX: Write handler error to redirected os.Stderr
				if err := handler(args); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}

				wStdout.Close()
				wStderr.Close()
				<-doneOut
				<-doneErr
				os.Stdout = origStdout
				os.Stderr = origStderr
			}()
		} else {
			exe := findExecutable(tokens[0])
			if exe == "" {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", tokens[0])
				continue
			}
			cmdArgs := tokens[1:]
			if redirectIdx != -1 {
				cmdArgs = tokens[1:redirectIdx]
			} else if errorRedirectIdx != -1 {
				cmdArgs = tokens[1:errorRedirectIdx]
			}
			cmd := exec.Command(exe, cmdArgs...)
			cmd.Args[0] = tokens[0] // Set argv[0] to the user-typed command
			cmd.Stdout = &outBuf
			cmd.Stderr = &errBuf
			cmd.Stdin = os.Stdin
			cmd.Run()
		}

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
