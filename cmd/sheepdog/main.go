package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/model"
)

func main() {
	conf, err := config.LoadConfig(".sheepdog.json")
	if err != nil {
		panic(err)
	}

	m := model.NewModel(conf)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Printf("Whoops, there was an error: %v\n", err)
		os.Exit(1)
	}
}
