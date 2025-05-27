package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func parseQuotes(input string) []string {
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

func main() {
	builtins := map[string]func([]string) error{
		"exit": func(args []string) error {
			os.Exit(0)
			return nil
		},
		"pwd": func(args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("pwd: %v", err)
			}
			fmt.Println(cwd)
			return nil
		},
		"cd": func(args []string) error {
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
		},
		"echo": func(args []string) error {
			fmt.Println(strings.Join(args[1:], " "))
			return nil
		},
		"type": func(args []string) error {
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
		},
	}

	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}
		command = strings.TrimRight(command, "\r\n")

		tokens := parseQuotes(command)
		if len(tokens) == 0 {
			continue
		}

		redirectIdx, errorRedirectIdx := -1, -1
		for i, t := range tokens {
			if t == ">" || t == "1>" {
				redirectIdx = i
			} else if t == "2>" {
				errorRedirectIdx = i
			}
		}

		var outBuf, errBuf bytes.Buffer
		var handlerErr error

		if handler, ok := builtins[tokens[0]]; ok {
			args := tokens
			if redirectIdx != -1 {
				args = tokens[:redirectIdx]
			}
			// Capture output
			stdout := os.Stdout
			stderr := os.Stderr
			os.Stdout = &outBuf
			os.Stderr = &errBuf
			handlerErr = handler(args)
			os.Stdout = stdout
			os.Stderr = stderr
		} else {
			exe := findExecutable(tokens[0])
			if exe == "" {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", tokens[0])
				continue
			}
			cmdArgs := tokens[1:]
			if redirectIdx != -1 {
				cmdArgs = tokens[1:redirectIdx]
			}
			cmd := exec.Command(exe, cmdArgs...)
			cmd.Stdout = &outBuf
			cmd.Stderr = &errBuf
			cmd.Stdin = os.Stdin
			handlerErr = cmd.Run()
		}

		// Handle stdout redirection
		if redirectIdx != -1 && redirectIdx+1 < len(tokens) {
			f, err := os.Create(tokens[redirectIdx+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[redirectIdx+1], err)
			} else {
				f.Write(outBuf.Bytes())
				f.Close()
			}
		} else {
			os.Stdout.Write(outBuf.Bytes())
		}

		// Handle stderr redirection
		if errorRedirectIdx != -1 && errorRedirectIdx+1 < len(tokens) {
			f, err := os.Create(tokens[errorRedirectIdx+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[errorRedirectIdx+1], err)
			} else {
				f.Write(errBuf.Bytes())
				f.Close()
			}
		} else {
			os.Stderr.Write(errBuf.Bytes())
		}

		if handlerErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", handlerErr)
		}
	}
}
