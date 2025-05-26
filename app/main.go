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
		c := input[i]

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
		case ' ':
			if inSingleQuotes || inDoubleQuotes {
				buf.WriteByte(ch)
			} else if buf.Len() > 0 {
				tokens = append(tokens, buf.String())
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

		if len(command) >= 5 && command[:5] == "echo " {
			// Use parseQuotes to handle single quotes and adjacent quoted args
			args := parseQuotes(command[5:])
			fmt.Println(strings.Join(args, " "))
			continue
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

		// Try to execute external command if found in PATH
		tokens := parseQuotes(command)
		if len(tokens) > 0 {
			exe := findExecutable(tokens[0])
			if exe != "" {
				cmd := exec.Command(exe, tokens[1:]...)
				cmd.Args[0] = tokens[0]
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[0], err)
				}
				continue
			}
		}

		fmt.Println(command + ": command not found")
	}
}
