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
		case "":
			return
		default:
			fmt.Println(inpArgs[0] + ": command not found")
		}
	}
}
