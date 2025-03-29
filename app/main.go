package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type CommandSplit struct {
	inpArgs    []string
	descriptor string
	outputFile string
}

type OutputWriter struct {
	stdout io.Writer
	stderr io.Writer
	file   *os.File
}

func NewOutputWriter(redirect, outfile string) (*OutputWriter, error) {
	ow := &OutputWriter{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	var file *os.File
	var err error
	if outfile != "" {
		file, err = os.Create(outfile)
		if err != nil {
			return ow, err
		}
	}
	ow.file = file
	if redirect == "2>" {
		ow.stderr = file
	} else if redirect == "1>" || redirect == ">" {
		ow.stdout = file
	}
	return ow, nil
}

// WriteOutput writes output to the correct destination
func (ow *OutputWriter) WriteOutput(outMsg, errMsg string) {
	fmt.Fprint(ow.stderr, errMsg)
	fmt.Fprint(ow.stdout, outMsg)
}

func (ow *OutputWriter) Close() {
	if ow.file != nil {
		ow.file.Close()
	}
}

func parseInput(inp string) (inpCmdSplit CommandSplit) {
	inpCmdSplit = CommandSplit{inpArgs: []string{""}}

	inpArgs := []string{}
	if inp == "" {
		return
	}
	var inDQuotes, inQuotes, escaped bool = false, false, false
	var current strings.Builder

	for i := range inp {
		char := inp[i]
		if escaped {
			escaped = false
			if inDQuotes && !(char == '$' || char == '`' || char == '"' || char == '\\') {
				current.WriteByte('\\')
			}
			current.WriteByte(char)
			continue
		}

		switch char {
		case '\\':
			escaped = !inQuotes
			if !escaped {
				current.WriteByte('\\')
			}
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
			if current.Len() != 0 {
				inpArgs = append(inpArgs, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(char)
		}
	}
	if current.Len() > 0 {
		inpArgs = append(inpArgs, current.String())
	}

	inpCmdSplit.inpArgs = inpArgs
	for i, arg := range inpArgs {
		if arg == ">" || arg == "1>" || arg == "2>" {
			inpCmdSplit.descriptor = "1>"
			if arg == "2>" {
				inpCmdSplit.descriptor = arg
			}
			inpCmdSplit.outputFile = inpArgs[i+1]
			inpCmdSplit.inpArgs = inpArgs[:i]
			break
		}
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

		parsedInput := parseInput(command)
		inpArgs := parsedInput.inpArgs

		var outputMessage, errorMessage string
		switch inpArgs[0] {
		case "exit":
			stsCode, err := strconv.Atoi(inpArgs[1])
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(stsCode)
		case "echo":
			outputMessage = fmt.Sprintln(strings.Join(inpArgs[1:], " "))
		case "pwd":
			wd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			outputMessage = fmt.Sprintln(wd)
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
				errorMessage = fmt.Sprintln("cd: " + inpArgs[1] + ": No such file or directory")
			}
		case "type":
			cmd := inpArgs[1]
			builtInCommands := map[string]bool{"exit": true, "echo": true, "type": true, "pwd": true}
			if builtInCommands[cmd] {
				outputMessage = fmt.Sprintln(cmd + " is a shell builtin")
			} else {
				// This logic can be replaced by using os/exec#LookUpPath method
				pathDirs := strings.Split(os.Getenv("PATH"), ":")
				var found bool
				for _, dir := range pathDirs {
					if _, err = os.Stat(dir + "/" + cmd); err == nil {
						found = true
						outputMessage = fmt.Sprintln(cmd + " is " + dir + "/" + cmd)
						break
					}
				}
				if !found {
					errorMessage = fmt.Sprintln(cmd + ": not found")
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
					errorMessage = errbuf.String()
				}
				outputMessage = outbuf.String()
			}
		}

		writer, err := NewOutputWriter(parsedInput.descriptor, parsedInput.outputFile)
		if err != nil {
			log.Fatal(err)
		}
		defer writer.Close()
		writer.WriteOutput(outputMessage, errorMessage)
	}
}
