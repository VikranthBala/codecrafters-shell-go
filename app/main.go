package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func parseInput(inp string) (inpArgs []string) {
	inpArgs = []string{}
	if inp == "" {
		return []string{""}
	}

	var inDQuotes, inQuotes, escaped bool = false, false, false
	var current strings.Builder

	for i := range inp {
		char := inp[i]
		if escaped {
			current.WriteByte(char)
			escaped = false
			continue
		}

		switch char {
		case '\\':
			escaped = true
		case '"':
			inDQuotes = !inQuotes && !inDQuotes
			if inQuotes {
				current.WriteByte('"')
			}
		case '\'':
			inQuotes = !inDQuotes && !inQuotes
			if inDQuotes {
				current.WriteByte('\'')
			}
		case ' ':
			if inQuotes || inDQuotes {
				current.WriteByte(' ')
				continue
			}
			inpArgs = append(inpArgs, current.String())
			current.Reset()
		default:
			current.WriteByte(char)
		}
	}
	if current.Len() > 0 {
		inpArgs = append(inpArgs, current.String())
	}
	return
}

func main() {

	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		command = strings.TrimRight(command, "\n")

		inpArgs := parseInput(command)

		switch inpArgs[0] {
		case "exit":
			stsCode, err := strconv.Atoi(inpArgs[1])
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(stsCode)
		case "echo":
			fmt.Println(strings.Join(inpArgs[1:], " "))
		case "pwd":
			wd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(wd)
		case "cd":
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			if len(inpArgs) == 1 || inpArgs[1] == "~" {
				err = os.Chdir(homeDir)
			} else {
				err = os.Chdir(inpArgs[1])
			}
			if err != nil {
				fmt.Println("cd: " + inpArgs[1] + ": No such file or directory")
			}
		case "type":
			cmd := inpArgs[1]
			builtInCommands := map[string]bool{"exit": true, "echo": true, "type": true, "pwd": true}
			if builtInCommands[cmd] {
				fmt.Println(cmd + " is a shell builtin")
			} else {
				// This logic can be replaced by using os/exec#LookUpPath method
				pathDirs := strings.Split(os.Getenv("PATH"), ":")
				var found bool
				for _, dir := range pathDirs {
					if _, err = os.Stat(dir + "/" + cmd); err == nil {
						found = true
						fmt.Println(cmd + " is " + dir + "/" + cmd)
						break
					}
				}
				if !found {
					fmt.Println(cmd + ": not found")
				}
			}
		case "":
			return
		default:
			if binPath, err := exec.LookPath(inpArgs[0]); err != nil {
				fmt.Println(inpArgs[0] + ": command not found")
			} else {
				cmd := exec.Command(inpArgs[0], inpArgs[1:]...)
				cmd.Dir = filepath.Dir(binPath)
				var outbuf, errbuf bytes.Buffer
				cmd.Stdout = &outbuf
				cmd.Stderr = &errbuf
				err := cmd.Run()
				if err != nil {
					log.Println("stderr: ", errbuf.String())
					log.Fatal(err)
				}
				fmt.Printf("%s", outbuf.String())
			}
		}
	}
}
