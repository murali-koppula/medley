package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	fileFlag := flag.String("f", "", "Path to the media layout yaml specification")
	flag.Parse()

	if *fileFlag == "" {
		fmt.Println("Error: Asset file path argument (-f) required.")
		os.Exit(1)
	}

	// 1. Convert structural content via Go YAML parsing
	tasks, err := ParseMediaFile(*fileFlag)
	if err != nil {
		fmt.Printf("Initialization Failure parsing configuration matrix: %v\n", err)
		os.Exit(1)
	}

	logChan := make(chan string)
	errChan := make(chan error)

	m := initialModel(tasks, logChan, errChan)
	p := tea.NewProgram(m)

	// 2. Fire the core processing pipeline safely into a background channel loop
	go func() {
		for i, task := range tasks {
			// Safely notify the TUI loop of the index progression
			// Signal progress index safely through your existing statusMsg channel
			p.Send(statusMsg(fmt.Sprintf("%d|Processing item: %s", i+1, task.Track.Title)))

			if err := ProcessTask(task, logChan); err != nil {
				errChan <- err
			}
		}
		p.Send(doneMsg{})
	}()

	// 3. Hand control over to the interactive terminal UI drawing context
	if _, err := p.Run(); err != nil {
		fmt.Printf("TUI Error: %v\n", err)
		os.Exit(1)
	}
}
