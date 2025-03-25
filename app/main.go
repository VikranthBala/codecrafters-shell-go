package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// func lookupCommandInPath(cmd string, pathD[])

func main() {

	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		command = strings.TrimRight(command, "\n")

		inpArgs := strings.Split(command, " ")

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
			err = os.Chdir(inpArgs[1])
			if err != nil {
				fmt.Println("cd: " + inpArgs[1] + " No such file or directory")
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
				out, err := cmd.Output()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("%s", string(out))
			}
		}
	}
}
