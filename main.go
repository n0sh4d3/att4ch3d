package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type (
	errMsg error
)

type model struct {
	textInput      textinput.Model
	borderStyle    lipgloss.Style
	containerStyle lipgloss.Style
	err            error
}

func initialModel() model {
	exec.Command("clear")
	width, height, _ := term.GetSize(1)

	ti := textinput.New()
	ti.Placeholder = "ask att4ch3d"
	ti.Focus()
	ti.CharLimit = 312
	ti.Width = 156

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("31")).
		Padding(1, 1).
		Width(44).
		Align(lipgloss.Center)

	containerStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		PaddingTop(height/8).
		Align(lipgloss.Center, lipgloss.Top)

	return model{
		textInput:      ti,
		borderStyle:    borderStyle,
		containerStyle: containerStyle,
		err:            nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			fmt.Println(m.textInput.Value())
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	content := m.borderStyle.Render(m.textInput.View())

	return m.containerStyle.Render(content)
}
