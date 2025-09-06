// Command sheepdog is the entry point for the Sheepdog application.
package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/model"
)

func main() {
	title := "sheepdog"
	fmt.Printf("\033]0;%s\007", title)

	f, err := os.OpenFile(".sheepdog.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	defer f.Close()

	handler := slog.NewTextHandler(f, nil)
	slog.SetDefault(slog.New(handler))

	conf, err := config.LoadConfig(".sheepdog.json")
	if err != nil {
		panic(err)
	}

	m := model.NewModel(conf)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Printf("Whoops, there was an error: %v\n", err)
		os.Exit(1)
	}

	m.CleanUp()
}
