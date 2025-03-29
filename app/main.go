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
	"sort"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var BUILT_IN_COMMANDS = []string{
	"exit",
	"echo",
	"type",
	"pwd",
	"cd",
}

var autoCompleteTurn = 0
var skipInput = false

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

// Can probably use this as a way to not look up in path again, but not implemented now
// var autoFilledCustomExecutablePath = ""

// removeDuplicatesAndSort removes duplicates from a slice and returns a sorted string slice
func removeDuplicatesAndSort(slice []string) []string {
	unique := make(map[string]struct{}) // Use map to track unique values
	var result []string

	for _, s := range slice {
		if _, exists := unique[s]; !exists {
			unique[s] = struct{}{}
			result = append(result, s)
		}
	}

	sort.Strings(result) // Sort the slice alphabetically
	return result
}

func getLongestCommonSubstring(suggestions []string) string {
	longestSubString := suggestions[0]
	lenLongestSubString := len(suggestions[0])
	for i := 1; i < len(suggestions); i++ {
		if lenLongestSubString > len(suggestions[i]) {
			lenSubStr := len(suggestions[i])
			subStr := suggestions[i]
		inner1:
			for lenSubStr > 0 {
				if strings.HasPrefix(longestSubString, subStr) {
					lenLongestSubString = lenSubStr
					longestSubString = subStr
					break inner1
				}
				subStr = subStr[:lenSubStr]
			}
		} else if lenLongestSubString < len(suggestions[i]) {
		inner2:
			for lenLongestSubString > 0 {
				if strings.HasPrefix(suggestions[i], longestSubString) {
					break inner2
				}
				longestSubString = longestSubString[:lenLongestSubString]
				lenLongestSubString--
			}
		} else {
		inner3:
			for j, longSub := range longestSubString {
				if byte(longSub) != suggestions[i][j] {
					longestSubString = longestSubString[:j]
					break inner3
				}
			}
		}
	}
	return longestSubString
}

func autoComplete(inp string) []string {

	matches := []string{}
	for _, cmd := range BUILT_IN_COMMANDS {
		if ok := strings.HasPrefix(cmd, inp); ok {
			matches = append(matches, cmd)
		}
	}

	for _, dir := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		fileName, err := filepath.Glob(dir + "/" + inp + "*")
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range fileName {
			matches = append(matches, filepath.Base(file))
		}
	}
	return removeDuplicatesAndSort(matches)
}

// Review this code again
func GetInputFromTerm(preInp string) (input string) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	input += preInp

	r := bufio.NewReader(os.Stdin)
loop:
	for {
		inp, _, err := r.ReadRune()
		if err != nil {
			fmt.Println(err)
			continue
		}

		switch inp {
		case '\x03':
			os.Exit(0)
		case '\r', '\n':
			fmt.Fprint(os.Stdout, "\r\n")
			break loop
		case '\x7F':
			if length := len(input); length > 0 {
				input = input[:length-1]
				fmt.Fprint(os.Stdout, "\b \b")
			}
		case '\t':
			suggestions := autoComplete(input)
			if len(suggestions) == 0 {
				fmt.Fprintf(os.Stdout, "\a")
			} else if len(suggestions) == 1 {
				suffix, _ := strings.CutPrefix(suggestions[0], input)
				input += suffix + " "
				fmt.Fprintf(os.Stdout, "%s", suffix+" ")
			} else {
				// HACK HACK HACK: This auto complete handling is broken fix this
				if autoCompleteTurn%2 == 1 {
					fmt.Fprintf(os.Stdout, "\r\n%s", strings.Join(suggestions, "  "))
					skipInput = true
					autoCompleteTurn = 0 // Reset the counter after displaying suggestions
					break loop
				}

				if commonCmdStr := getLongestCommonSubstring(suggestions); commonCmdStr != input {
					suffix, _ := strings.CutPrefix(commonCmdStr, input)
					input += suffix
					fmt.Fprintf(os.Stdout, "%s", suffix)
				} else {
					fmt.Fprintf(os.Stdout, "\a")
					autoCompleteTurn++
				}
			}
		default:
			input += string(inp)
			fmt.Fprint(os.Stdout, string(inp))
		}
	}
	return input
}

func NewOutputWriter(redirect, outfile string) (*OutputWriter, error) {
	ow := &OutputWriter{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	var file *os.File
	var err error
	if outfile != "" {
		flag := os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		if redirect == "2>>" || redirect == "1>>" {
			flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
		}
		file, err = os.OpenFile(outfile, flag, 0666)
		if err != nil {
			return ow, err
		}
	}
	ow.file = file
	if redirect == "2>" || redirect == "2>>" {
		ow.stderr = file
	} else if redirect == "1>" || redirect == "1>>" {
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
		if arg == ">" || arg == "1>" || arg == "2>" || arg == ">>" || arg == "1>>" || arg == "2>>" {
			switch arg {
			case ">", "1>":
				inpCmdSplit.descriptor = "1>"
			case "2>":
				inpCmdSplit.descriptor = "2>"
			case ">>", "1>>":
				inpCmdSplit.descriptor = "1>>"
			case "2>>":
				inpCmdSplit.descriptor = "2>>"
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
		fmt.Fprint(os.Stdout, "\r$ ")
		// command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		// if err != nil {
		// 	log.Fatal(err)
		// }

		command := GetInputFromTerm("")
		for skipInput {
			skipInput = false
			fmt.Fprint(os.Stdout, "\r\n$ "+command)
			command = GetInputFromTerm(command)
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
					if _, err := os.Stat(dir + "/" + cmd); err == nil {
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
