package main

import (
	"bufio"
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

// Helper to run an external command with optional output redirection
func runExternalCommand(exe string, tokens []string, stdoutOverride *os.File) error {
	cmd := exec.Command(exe, tokens[1:]...)
	cmd.Args[0] = tokens[0] // Use the user-typed command
	if stdoutOverride != nil {
		cmd.Stdout = stdoutOverride
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Helper function to handle output redirection
func handleOutputRedirection(tokens []string, redirectIdx int) (*os.File, error) {
	if redirectIdx != -1 && redirectIdx+1 < len(tokens) {
		f, err := os.Create(tokens[redirectIdx+1])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", tokens[redirectIdx+1], err)
		}
		return f, nil
	}
	return os.Stdout, nil
}

// Helper function to handle error redirection
func handleErrorRedirection(tokens []string, errorRedirectIdx int) (*os.File, error) {
	if errorRedirectIdx != -1 && errorRedirectIdx+1 < len(tokens) {
		f, err := os.Create(tokens[errorRedirectIdx+1])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", tokens[errorRedirectIdx+1], err)
		}
		return f, nil
	}
	return os.Stderr, nil
}

func main() {
	// Whitelist of valid builtins
	builtins := map[string]bool{
		"exit": true,
		"echo": true,
		"type": true,
		"pwd":  true,
		"cd":   true,
	}

	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}
		// Remove trailing newline
		command = command[:len(command)-1]

		if command == "exit" || command == "exit 0" {
			os.Exit(0)
		}

		if len(command) >= 3 && command[:3] == "cd " {
			tokens := strings.Fields(command)
			if len(tokens) != 2 {
				fmt.Println("cd: too many arguments")
				continue
			}
			arg := tokens[1]
			// Handle tilde as home directory
			if arg == "~" {
				home, err := os.UserHomeDir()
				if err != nil {
					fmt.Println("cd: cannot determine home directory")
					continue
				}
				arg = home
			}
			absPath, err := os.Stat(arg)
			if err != nil || !absPath.IsDir() {
				fmt.Printf("cd: %s: No such file or directory\n", arg)
				continue
			}
			// Change directory
			if err := os.Chdir(arg); err != nil {
				fmt.Printf("cd: %s: No such file or directory\n", arg)
				continue
			}
			continue
		}

		if command == "pwd" {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Printf("pwd: %v\n", err)
			} else {
				fmt.Println(cwd)
			}
			continue
		}

		// Handle "type" command
		if len(command) >= 5 && command[:5] == "type " {
			tokens := strings.Fields(command)
			if len(tokens) != 2 {
				fmt.Println("type: too many arguments")
				continue
			}
			arg := tokens[1]
			if builtins[arg] {
				fmt.Printf("%s is a shell builtin\n", arg)
			} else {
				fullPath := findExecutable(arg)
				if fullPath != "" {
					fmt.Printf("%s is %s\n", arg, fullPath)
				} else {
					fmt.Printf("%s: not found\n", arg)
				}
			}
			continue
		}

		// Try to execute external command or echo, with or without redirection
		tokens := parseQuotes(command)
		if len(tokens) > 0 {
			redirectIdx := -1
			errorRedirectIdx := -1
			for i, t := range tokens {
				if t == ">" || t == "1>" {
					redirectIdx = i
					break
				} else if t == "2>" {
					errorRedirectIdx = i
					break
				}
			}

			// Handle echo with or without output redirection
			if tokens[0] == "echo" {
				out, err := handleOutputRedirection(tokens, redirectIdx)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				defer out.Close()

				errOut, err := handleErrorRedirection(tokens, errorRedirectIdx)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				defer errOut.Close()

				args := tokens[1:]
				if redirectIdx != -1 {
					args = tokens[1:redirectIdx]
				}

				fmt.Fprintln(out, strings.Join(args, " "))
				continue
			}

			// Handle external commands with redirection
			exe := findExecutable(tokens[0])
			if exe != "" {
				out, err := handleOutputRedirection(tokens, redirectIdx)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				defer out.Close()

				errOut, err := handleErrorRedirection(tokens, errorRedirectIdx)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				defer errOut.Close()

				cmd := exec.Command(exe, tokens[1:]...)
				cmd.Stdout = out
				cmd.Stderr = errOut
				cmd.Stdin = os.Stdin

				if err := cmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[0], err)
				}
				continue
			}

			fmt.Println(command + ": command not found")
		}
	}
}
