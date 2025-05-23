package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	// Whitelist of valid builtins
	builtins := map[string]bool{
		"exit": true,
		"echo": true,
		"type": true,
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
			fmt.Println(command[5:])
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
				// Check PATH for executable
				pathEnv := os.Getenv("PATH")
				paths := strings.Split(pathEnv, string(os.PathListSeparator))
				found := false
				for _, dir := range paths {
					fullPath := dir + string(os.PathSeparator) + arg
					if _, err := os.Stat(fullPath); err == nil {
						fmt.Printf("%s is %s\n", arg, fullPath)
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("%s: not found\n", arg)
				}
			}
			continue
		}

		fmt.Println(command + ": command not found")
	}
}
