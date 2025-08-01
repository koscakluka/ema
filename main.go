package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"os"
	"strings"
	"sync"
	"time"

	"github.com/koscakluka/ema/core"
	"github.com/koscakluka/ema/internal/utils"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

const (
	sidebarWidth      = 33
	sidebarPadding    = 1
	sidebarOuterWidth = sidebarWidth + sidebarPadding*2

	viewportPadding = 1
)

type speechDetectedMsg bool
type endRecordingMsg struct{}

type responseMsg struct {
	text   string
	buffer bool
}
type interimTranscriptMsg string

var program *tea.Program

var output strings.Builder
var mutex sync.RWMutex

type model struct {
	orchestrator *orchestration.Orchestrator
	buffer       *bytes.Buffer
	buffering    *bool

	termWidth         int
	termHeight        int
	ready             bool
	speechDetected    bool
	interimTranscript string

	viewport        viewport.Model
	automaticScroll bool

	endRecordingTimer *time.Timer
}

func (m model) Init() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := os.MkdirAll("tmp", 0755); err != nil {
			program.Quit()
			return nil
		}
		f, err := os.OpenFile("tmp/log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			program.Quit()
			return nil
		}
		os.Stdout = f
		log.SetOutput(f)
		fmt.Println("redirected stdout")
		return nil
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	viewportHeight := m.termHeight - viewportPadding*2 - 3

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height

		viewportHeight = m.termHeight - viewportPadding*2 - 3
		if !m.ready {
			m.viewport = viewport.New(m.viewportWidth(), viewportHeight)
			m.ready = true
		} else {
			m.viewport.Width = m.viewportWidth()
			m.viewport.Height = viewportHeight
		}
		m.viewport.SetContent(m.getContent())
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			m.orchestrator.IsRecording = true
			if m.endRecordingTimer != nil {
				if !m.endRecordingTimer.Stop() {
					select {
					case <-m.endRecordingTimer.C:
						return m, func() tea.Msg { return endRecordingMsg{} }
					default:
					}
				}
			}
			m.endRecordingTimer = time.NewTimer(time.Millisecond * 100)

			return m, func() tea.Msg {
				<-m.endRecordingTimer.C
				return endRecordingMsg{}
			}

		case "l":
			m.orchestrator.AlwaysRecording = !m.orchestrator.AlwaysRecording

		case "m":
			m.orchestrator.IsSpeaking = !m.orchestrator.IsSpeaking

		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case endRecordingMsg:
		m.orchestrator.IsRecording = false
		m.endRecordingTimer = nil
		return m, nil

	case interimTranscriptMsg:
		m.interimTranscript = string(msg)
		m.viewport.SetContent(m.getContent())

	case speechDetectedMsg:
		m.speechDetected = bool(msg)

	case responseMsg:
		if msg.buffer {
			m.buffer.WriteString(string(msg.text))
			if !*m.buffering {
				*m.buffering = true
				return m, tea.Cmd(func() tea.Msg {
					defer func() {
						*m.buffering = false
					}()
					for {
						b, err := m.buffer.ReadByte()
						time.Sleep(time.Millisecond * 10)
						if err == io.EOF {
							return nil
						} else if err != nil {
							// TODO: handle error
							return nil
						}
						program.Send(responseMsg{text: string(b), buffer: false})
					}
				})
			}
		} else {
			mutex.Lock()
			output.WriteString(string(msg.text))
			m.viewport.SetContent(m.getContent())
			mutex.Unlock()
			if m.automaticScroll {
				m.viewport.GotoBottom()
			}
		}
	}

	m.viewport, _ = m.viewport.Update(msg)
	if m.viewport.AtBottom() {
		m.automaticScroll = true
	} else {
		m.automaticScroll = false
	}

	return m, nil
}

func (m model) viewportWidth() int {
	return m.termWidth - sidebarOuterWidth - viewportPadding*2
}

func (m model) getContent() string {
	output := strings.TrimSpace(output.String())
	if m.interimTranscript != "" {
		output += "\n" + strings.TrimSpace(m.interimTranscript)
	}
	return wordwrap.String(output, m.viewportWidth()-4)
}

func (m model) View() string {
	if m.termWidth == 0 {
		return "Loading..."
	}

	mainStyle := lipgloss.NewStyle().
		Padding(1).
		Width(m.termWidth - sidebarOuterWidth).
		Height(m.termHeight - 3)

	sidebarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(sidebarPadding).
		Width(sidebarWidth).
		Height(m.termHeight - 2)

	mainContent := mainStyle.Render(m.viewport.View())

	sidebarLabelStyle := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("220"))
	sidebarValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	sidebarContent := []string{}
	for _, line := range []struct {
		label string
		value string
	}{
		{label: "Recording", value: fmt.Sprintf("%v", m.orchestrator.IsRecording || m.orchestrator.AlwaysRecording)},
		{label: "Automatic Scroll", value: fmt.Sprintf("%v", m.automaticScroll)},
		{label: "Speaking", value: fmt.Sprintf("%v", m.speechDetected)},
	} {
		sidebarContent = append(sidebarContent,
			fmt.Sprintf("%s: %v",
				sidebarLabelStyle.Render(line.label),
				sidebarValueStyle.Render(line.value),
			),
		)
	}
	sidebar := sidebarStyle.Render(strings.Join(sidebarContent, "\n"))

	footer := lipgloss.NewStyle().
		PaddingTop(1).
		Foreground(lipgloss.Color("241")).
		Render("Press 'q' or 'Ctrl+C' to quit")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			mainContent,
			footer,
		),
		sidebar,
	)
}

func main() {
	orchestrator := orchestration.NewOrchestrator()

	program = tea.NewProgram(
		model{
			automaticScroll: true,
			orchestrator:    orchestrator,
			buffer:          bytes.NewBuffer([]byte{}),
			buffering:       utils.Ptr(false),
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go orchestrator.ListenForSpeech(ctx, orchestration.Callbacks{
		OnTranscription: func(transcript string) {
			program.Send(responseMsg{text: transcript})
		},
		OnInterimTranscription: func(transcript string) {
			program.Send(interimTranscriptMsg(transcript))
		},
		OnSpeakingStateChanged: func(isSpeaking bool) {
			program.Send(speechDetectedMsg(isSpeaking))
		},
		OnResponse: func(response string) {
			program.Send(responseMsg{text: response, buffer: true})
		},
	})

	if _, err := program.Run(); err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
