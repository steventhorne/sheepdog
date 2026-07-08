// Command sheepdog is the entry point for the Sheepdog application.
package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/model"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	// Installed via `go install module@version` — the module version is
	// recorded in the build info even without ldflags.
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	title := "Sheepdog"
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

	m := model.NewModel(conf, resolveVersion())
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Printf("Whoops, there was an error: %v\n", err)
		os.Exit(1)
	}
}
