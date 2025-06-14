// f
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
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

const (
	SpeechLen = 20
	// fucjing lipgloss colors
	Quiet    = "#7aa2f7"
	MidQuiet = "#7dcfff"
	MidLoud  = "#b4f9f8"
	Loud     = "#c0caf5"
)

var speechElems = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

type (
	errMsg    error
	tickMsg   time.Time
	volumeMsg float64
)

type VolumeStreamer struct {
	streamer    beep.Streamer
	volume      *effects.Volume
	volumeLevel float64
	callback    func(float64)
}

func NewVolumeStreamer(s beep.Streamer, callback func(float64)) *VolumeStreamer {
	volume := &effects.Volume{
		Streamer: s,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	return &VolumeStreamer{
		streamer: s,
		volume:   volume,
		callback: callback,
	}
}

func (vs *VolumeStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = vs.streamer.Stream(samples)

	if n > 0 {
		var sum float64
		// lmafo
		for i := range n {
			sample := (samples[i][0] + samples[i][1]) / 2
			sum += sample * sample
		}
		rms := math.Sqrt(sum / float64(n))

		vs.volumeLevel = vs.volumeLevel*0.7 + rms*0.3

		if vs.callback != nil {
			normalizedVolume := math.Min(vs.volumeLevel*10, 1.0)
			vs.callback(normalizedVolume)
		}
	}

	return n, ok
}

func (vs *VolumeStreamer) Err() error {
	return vs.streamer.Err()
}

type model struct {
	textInput      textinput.Model
	borderStyle    lipgloss.Style
	containerStyle lipgloss.Style
	barHeights     []int
	barMomentum    []float64
	barTargets     []float64
	isSpeaking     bool
	audioFinished  bool // NEW: track when audio is done but bars are still fading
	aiSpeech       string
	aiSpeechStyles []lipgloss.Style
	tickCount      int
	speakingStart  time.Time
	currentVolume  float64
	volumeHistory  []float64
	program        *tea.Program
	randomSeeds    []float64
	err            error
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m)
	m.program = p

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
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

	aiStyle := []lipgloss.Style{DefaultSpeech, LoudSpeech, QuietSpeech, MidLoudSpeech, MidQuietSpeech}

	initialBars := make([]int, SpeechLen)
	initialMomentum := make([]float64, SpeechLen)
	initialTargets := make([]float64, SpeechLen)
	initialSeeds := make([]float64, SpeechLen)
	initialSpeech := strings.Repeat(speechElems[0], SpeechLen)

	for i := range initialSeeds {
		initialSeeds[i] = rand.Float64() * 100
	}

	return model{
		textInput:      ti,
		borderStyle:    borderStyle,
		containerStyle: containerStyle,
		err:            nil,
		aiSpeech:       initialSpeech,
		aiSpeechStyles: aiStyle,
		barHeights:     initialBars,
		barMomentum:    initialMomentum,
		barTargets:     initialTargets,
		isSpeaking:     false,
		audioFinished:  false, // NEW: initialize to false
		tickCount:      0,
		currentVolume:  0.0,
		volumeHistory:  make([]float64, 10),
		randomSeeds:    initialSeeds,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case volumeMsg:
		m.currentVolume = float64(msg)

		copy(m.volumeHistory[1:], m.volumeHistory[0:])
		m.volumeHistory[0] = m.currentVolume

		if m.isSpeaking {
			return m, tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}

	case tickMsg:
		if m.isSpeaking || m.audioFinished {
			// Check if audio file is gone (playback finished)
			if _, err := os.Stat("file.mp3"); os.IsNotExist(err) {
				if m.isSpeaking {
					// Audio just finished, start fadeout
					m.isSpeaking = false
					m.audioFinished = true
					m.currentVolume = 0.0
				}
			}

			// Update bars first
			m.updateBarsFromVolume()

			// After updating, check if all bars have faded to 0
			if m.audioFinished {
				allBarsZero := true
				for _, height := range m.barHeights {
					if height > 0 {
						allBarsZero = false
						break
					}
				}

				// If all bars are at 0, stop the animation
				if allBarsZero {
					m.audioFinished = false
					m.textInput.SetValue("")
					return m, nil
				}
			}

			return m, tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if !m.isSpeaking && !m.audioFinished {
				m.startSpeaking()
				return m, tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
					return tickMsg(t)
				})
			}
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	if !m.isSpeaking && !m.audioFinished {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

func (m *model) startSpeaking() {
	m.isSpeaking = true
	m.audioFinished = false
	m.speakingStart = time.Now()

	cmd := exec.Command("python3", "tts.py", m.textInput.Value())
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error executing python command: %v\n", err)
		m.isSpeaking = false
		return
	}

	go m.playAudioWithVolumeMonitoring()
}

func (m *model) playAudioWithVolumeMonitoring() {
	file, err := os.Open("file.mp3")
	if err != nil {
		fmt.Printf("Error opening audio file: %v\n", err)
		m.isSpeaking = false
		return
	}
	defer file.Close()

	streamer, format, err := mp3.Decode(file)
	if err != nil {
		fmt.Printf("Error decoding MP3: %v\n", err)
		m.isSpeaking = false
		return
	}
	defer streamer.Close()

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		fmt.Printf("Error initializing speaker: %v\n", err)
		m.isSpeaking = false
		return
	}

	// start fucking wobbling when actual playback begins
	m.currentVolume = 0.5

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done

	file.Close()
	os.Remove("file.mp3")

	// the tick handler will detect the missing file and start the fadeout process
}

func (m *model) updateBarsFromVolume() {
	currentTime := float64(time.Now().UnixMilli())

	// if audio is finished, gradually fade all bars to 0
	if m.audioFinished {
		for i := range m.barHeights {
			if m.barHeights[i] > 0 {
				// mse momentum-based fadeout for smoother animation
				m.barTargets[i] = 0
				m.barMomentum[i] = m.barMomentum[i]*0.8 - 0.3 // fade momentum

				newHeight := float64(m.barHeights[i]) + m.barMomentum[i]
				if newHeight <= 0 {
					m.barHeights[i] = 0
					m.barMomentum[i] = 0
				} else {
					m.barHeights[i] = int(math.Round(newHeight))
				}
			}
		}
		return
	}

	// normal operation when audio is playing
	if !m.isSpeaking {
		return
	}

	// calculate average volume from recent history for stability
	var avgVolume float64
	for _, vol := range m.volumeHistory {
		avgVolume += vol
	}
	avgVolume /= float64(len(m.volumeHistory))

	// use both current and average volume for more realistic movement
	mixedVolume := (m.currentVolume*0.7 + avgVolume*0.3)

	if mixedVolume < 0.10 {
		mixedVolume = 0.2 + rand.Float64()*0.3
	} else {
		// moderate amplification for good range without maxing out
		mixedVolume = math.Min(mixedVolume*1.8, 0.85)
	}

	for i := range SpeechLen {
		// create frequency-based variation
		// Lower indices = lower frequencies, higher indices = higher frequencies
		frequencyWeight := 1.0
		if i < SpeechLen/3 {
			frequencyWeight = 0.8 + mixedVolume*0.6
		} else if i > 2*SpeechLen/3 {
			frequencyWeight = 0.7 + mixedVolume*0.9 + rand.Float64()*0.4
		} else {
			frequencyWeight = 0.75 + mixedVolume*0.75
		}

		phase := m.randomSeeds[i] + currentTime/200.0
		randomVariation := (math.Sin(phase) + math.Sin(phase*1.7) + math.Sin(phase*0.3)) / 3.0
		randomVariation *= 0.3

		var neighborInfluence float64
		if i > 0 {
			neighborInfluence += float64(m.barHeights[i-1]) * 0.1
		}
		if i < SpeechLen-1 {
			neighborInfluence += float64(m.barHeights[i+1]) * 0.1
		}

		baseHeight := mixedVolume * frequencyWeight * 10.5
		targetHeight := baseHeight + randomVariation*2.2 + neighborInfluence

		// could use max but, i just don't care
		if targetHeight < 0 {
			targetHeight = 0
		}
		if targetHeight > 9 {
			targetHeight = 9
		}

		m.barTargets[i] = targetHeight

		diff := m.barTargets[i] - float64(m.barHeights[i])
		m.barMomentum[i] = m.barMomentum[i]*0.7 + diff*0.4

		newHeight := float64(m.barHeights[i]) + m.barMomentum[i]

		// could use max but, i just don't care
		if newHeight < 0 {
			newHeight = 0
		}
		if newHeight > 9 {
			newHeight = 9
		}

		m.barHeights[i] = int(math.Round(newHeight))

		if rand.Float64() < 0.06 && mixedVolume > 0.25 {
			spike := int(rand.Float64() * 3)
			if m.barHeights[i]+spike <= 9 {
				m.barHeights[i] += spike
			}
		}
	}
}

func (m model) View() string {
	rows := 10

	QuietStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(Quiet))
	MidQuietStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(MidQuiet))
	MidLoudStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(MidLoud))
	LoudStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(Loud))

	getRowStyle := func(rowLevel int) lipgloss.Style {
		switch rowLevel {
		case 9, 8, 7:
			return LoudStyle
		case 6, 5, 4:
			return MidLoudStyle
		case 3, 2:
			return MidQuietStyle
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
