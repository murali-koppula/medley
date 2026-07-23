package main

import (
	"log"
	"os"

	"medley/cmd"
)

func main() {
	command := cmd.Command()
	args := os.Args[1:]

	_, _, err := command.Find(args)

	if err != nil && args[0] != "__complete" {
		command.Println(command.UsageString())
		os.Exit(0)
	}

	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error running %s. %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
