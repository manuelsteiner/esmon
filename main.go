package main

import (
	"esmon/tui"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(tui.NewMainModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal("Failed to start program: ", err)
	}
}
