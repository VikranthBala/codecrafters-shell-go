package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Fprint(os.Stdout, "$ ")

	command, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(command[:len(command)-1] + ": command not found")
}
