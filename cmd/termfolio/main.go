package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/badele/termfolio/internal/config"
	"github.com/badele/termfolio/internal/ui"
)

func main() {
	// Parse the optional config flag and start the TUI.
	configPath := flag.String("config", "", "Path to the YAML config file")
	flag.Parse()

	loadedConfig, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Configuration error: %v", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewModel(loadedConfig), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*config.Config, error) {
	// Load only when an explicit path is provided.
	if path == "" {
		return nil, fmt.Errorf("--config is required")
	}

	loaded, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return &loaded, nil
}
