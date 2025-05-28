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

func main() {
	builtins := make(map[string]func([]string) error)

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

	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}
		command = strings.TrimRight(command, "\r\n")

		tokens := parseMetas(command)
		if len(tokens) == 0 {
			continue
		}

		redirectIdx, errorRedirectIdx := -1, -1
		for i, t := range tokens {
			if t == ">" || t == "1>" {
				redirectIdx = i
				break
			} else if t == "2>" {
				errorRedirectIdx = i
				break
			}
		}

		var outBuf, errBuf bytes.Buffer

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
			cmd.Stdout = &outBuf
			cmd.Stderr = &errBuf
			cmd.Stdin = os.Stdin
			cmd.Run()
		}

		// Handle redirection (only the first redirect operator found)
		if redirectIdx != -1 && redirectIdx+1 < len(tokens) {
			f, err := os.Create(tokens[redirectIdx+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", tokens[redirectIdx+1], err)
			} else {
				f.Write(outBuf.Bytes())
				f.Close()
			}
			os.Stderr.Write(errBuf.Bytes())
		} else if errorRedirectIdx != -1 && errorRedirectIdx+1 < len(tokens) {
			f, err := os.Create(tokens[errorRedirectIdx+1])
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
