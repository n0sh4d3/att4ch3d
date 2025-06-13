package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

const (
	SpeechLen = 20

	// bubble tea lipgloss.Color values or how tf this is called
	Quiet    = "#7aa2f7"
	MidQuiet = "#7dcfff"
	MidLoud  = "#b4f9f8"
	Loud     = "#c0caf5"
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
	aiSpeechStyles []lipgloss.Style
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
		Foreground(lipgloss.Color(MidLoud)).
		Bold(true)

	LoudSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(Loud)).
		Bold(true)

	MidLoudSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(MidLoud)).
		Bold(true)

	MidQuietSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(MidQuiet)).
		Bold(true)

	QuietSpeech := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Padding(2, 4).
		Foreground(lipgloss.Color(Quiet)).
		Bold(true)

	aiStyle := []lipgloss.Style{}
	aiStyle = append(aiStyle, DefaultSpeech, LoudSpeech, QuietSpeech, MidLoudSpeech, MidQuietSpeech)

	initialBars := make([]int, SpeechLen)
	initialSpeech := strings.Repeat(speechElems[0], SpeechLen)

	return model{
		textInput:      ti,
		borderStyle:    borderStyle,
		containerStyle: containerStyle,
		err:            nil,
		aiSpeech:       initialSpeech,
		aiSpeechStyles: aiStyle,
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

			for i := range SpeechLen {
				// calculate distance from center (0.0 at edges, 1.0 at center)
				center := float64(SpeechLen-1) / 2.0
				distanceFromCenter := 1.0 - math.Abs(float64(i)-center)/center

				// base height follows mouth shape - higher in middle, but edges have minimum height
				baseHeight := int(1 + float64(4)*distanceFromCenter*distanceFromCenter) // 1-5 range instead of 0-5

				// add some randomness but keep it realistic
				variation := randRange(-1, 1)
				height := baseHeight + variation

				if height < 1 { // minimum height of 1 so edges never go to 0
					height = 1
				}
				if height >= 6 { // max height should match number of rows
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

			// lazy tts cuz i'm lazy hoe :3
			cmd := exec.Command("python3", "tts.py", m.textInput.Value())
			err := cmd.Start()
			if err != nil {
				fmt.Printf("Error executing python command: %v\n", err)
				m.isSpeaking = false
				return m, nil
			}

			go func() {
				time.Sleep(1 * time.Second)
				file, err := os.Open("file.mp3")
				if err != nil {
				}

				streamer, format, err := mp3.Decode(file)
				if err != nil {
					log.Fatal(err)
				}
				defer streamer.Close()

				err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
				if err != nil {
					log.Fatal(err)
				}

				done := make(chan bool)
				speaker.Play(beep.Seq(streamer, beep.Callback(func() {
					done <- true
				})))

				<-done
				file.Close()
			}()

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
	rows := 10
	const (
		Quiet    = "#7aa2f7"
		MidQuiet = "#7dcfff"
		MidLoud  = "#b4f9f8"
		Loud     = "#c0caf5"
	)

	QuietStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(Quiet))
	MidQuietStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(MidQuiet))
	MidLoudStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(MidLoud))
	LoudStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(Loud))

	getRowStyle := func(rowLevel int) lipgloss.Style {
		switch rowLevel {
		case 7, 6:
			return LoudStyle
		case 5, 4:
			return MidLoudStyle
		case 3, 2:
			return MidQuietStyle
		case 1, 0:
			return QuietStyle
		default:
			return QuietStyle
		}
	}

	var allRows []string
	for row := range rows {
		var rowParts []string
		barLevel := rows - 1 - row

		rowStyle := getRowStyle(barLevel)

		for _, height := range m.barHeights {
			if height > barLevel {
				char := strings.Repeat(speechElems[7], 3)
				rowParts = append(rowParts, char)
			} else {
				rowParts = append(rowParts, strings.Repeat(" ", 3))
			}
		}

		rowStr := strings.Join(rowParts, "")
		styledRow := rowStyle.Render(rowStr)
		allRows = append(allRows, styledRow)
	}

	m.aiSpeech = strings.Join(allRows, "\n")

	speechBars := m.aiSpeechStyles[2].Render(m.aiSpeech)
	content := m.borderStyle.Render(m.textInput.View())
	combined := lipgloss.JoinVertical(lipgloss.Center, content, "", speechBars)

	return m.containerStyle.Render(combined)
}

func randRange(min, max int) int {
	return rand.Intn(max-min+1) + min
}
