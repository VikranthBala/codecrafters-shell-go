package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

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
		case "type":
			builtInCommands := map[string]bool{"exit": true, "echo": true, "type": true}
			if builtInCommands[inpArgs[1]] {
				fmt.Println(inpArgs[1] + " is a shell builtin")
			} else {
				fmt.Println(inpArgs[1] + ": not found")
			}
		case "":
			return
		default:
			fmt.Println(inpArgs[0] + ": command not found")
		}
	}
}
