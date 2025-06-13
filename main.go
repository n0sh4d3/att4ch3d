package main

import (
	"log"
	"math"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

const (
	SpeechLen = 20

	// bubble tea lipgloss.Color values or how tf this is called
	LoudColor    = "51"
	DefaultColor = "31"
	QuietColor   = "4"
)

var speechElems = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

type (
	errMsg  error
	tickMsg time.Time
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	textInput      textinput.Model
	borderStyle    lipgloss.Style
	containerStyle lipgloss.Style
	barHeights     []int
	isSpeaking     bool
	aiSpeech       string
	aiSpeechStyle  []lipgloss.Style
	tickCount      int
	speakingStart  time.Time
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

	// this is just fucking sytling
	DefaultSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(DefaultColor)).
		Bold(true)

	LoudSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(LoudColor)).
		Bold(true)

	QuietSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(QuietColor)).
		Bold(true)

	aiStyle := []lipgloss.Style{}
	aiStyle = append(aiStyle, DefaultSpeech, LoudSpeech, QuietSpeech)

	initialBars := make([]int, SpeechLen)
	initialSpeech := strings.Repeat(speechElems[0], SpeechLen)

	return model{
		textInput:      ti,
		borderStyle:    borderStyle,
		containerStyle: containerStyle,
		err:            nil,
		aiSpeech:       initialSpeech,
		aiSpeechStyle:  aiStyle,
		barHeights:     initialBars,
		isSpeaking:     false,
		tickCount:      0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		if m.isSpeaking {
			if time.Since(m.speakingStart) > 3*time.Second {
				m.isSpeaking = false
				m.textInput.SetValue("")
				for i := range m.barHeights {
					m.barHeights[i] = 0
				}
				return m, nil
			}

			// Generate mouth-like pattern - higher in middle, lower at edges
			for i := range SpeechLen {
				// Calculate distance from center (0.0 at edges, 1.0 at center)
				center := float64(SpeechLen-1) / 2.0
				distanceFromCenter := 1.0 - math.Abs(float64(i)-center)/center

				// Base height follows mouth shape - higher in middle, but edges have minimum height
				baseHeight := int(1 + float64(4)*distanceFromCenter*distanceFromCenter) // 1-5 range instead of 0-5

				// Add some randomness but keep it realistic
				variation := randRange(-1, 1)
				height := baseHeight + variation

				if height < 1 { // Minimum height of 1 so edges never go to 0
					height = 1
				}
				if height >= 6 { // Max height should match number of rows
					height = 5
				}

				m.barHeights[i] = height
			}

			return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.isSpeaking = true
			m.speakingStart = time.Now()
			return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	if !m.isSpeaking {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	// Build multiple rows of speech bars to make it taller - stacked properly
	rows := 6 // Number of rows for height
	var allRows []string

	for row := range rows {
		var speechBuilder strings.Builder
		for _, height := range m.barHeights {
			// Calculate which character to show for this row
			// Bottom row (rows-1) shows full bar, top row (0) shows only if bar is tall enough
			barLevel := rows - 1 - row // Convert row to bar level (0 = top, rows-1 = bottom)

			var char string
			if height > barLevel {
				// Bar is tall enough to show at this level
				char = strings.Repeat(speechElems[7], 3) // Use full block for stacking effect
			} else {
				// Bar not tall enough, show spaces
				char = strings.Repeat(" ", 3)
			}
			speechBuilder.WriteString(char)
		}
		allRows = append(allRows, speechBuilder.String())
	}

	// Join all rows with newlines to create vertical stack
	m.aiSpeech = strings.Join(allRows, "\n")
	speechBars := m.aiSpeechStyle[0].Render(m.aiSpeech)

	content := m.borderStyle.Render(m.textInput.View())
	combined := lipgloss.JoinVertical(lipgloss.Center, content, "", speechBars)

	return m.containerStyle.Render(combined)
}

func randRange(min, max int) int {
	return rand.Intn(max-min+1) + min
}
