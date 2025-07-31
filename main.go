package main

import (
	"context"
	"fmt"
	"io"

	"os"
	"strings"
	"sync"
	"time"

	"github.com/koscakluka/ema/core"

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

type stdoutMsg string
type speakingMsg bool

type interimTranscriptMsg string
type endRecordingMsg struct{}

var program *tea.Program

var output strings.Builder
var mutex sync.RWMutex

type model struct {
	orchestrator *orchestration.Orchestrator

	termWidth         int
	termHeight        int
	ready             bool
	speaking          bool
	interimTranscript string

	viewport        viewport.Model
	automaticScroll bool

	endRecordingTimer *time.Timer
}

func (m model) Init() tea.Cmd {
	// Redirect output to the program
	return tea.Cmd(func() tea.Msg {
		var r *os.File
		var err error
		r, os.Stdout, err = os.Pipe()
		if err != nil {
			return nil
		}

		go func() {
			buffer := make([]byte, 1024)
			for {
				n, err := r.Read(buffer)
				if err != nil && err != io.EOF {
					break
				}
				for i := range n {
					program.Send(stdoutMsg(string(buffer[i : i+1])))
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()

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

	case speakingMsg:
		m.speaking = bool(msg)

	case stdoutMsg:
		mutex.Lock()
		output.WriteString(string(msg))
		m.viewport.SetContent(m.getContent())
		mutex.Unlock()
		if m.automaticScroll {
			m.viewport.GotoBottom()
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

	sidebar := sidebarStyle.Render(strings.Join([]string{
		fmt.Sprintf("%s: %v",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Recording"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%v", m.orchestrator.IsRecording || m.orchestrator.AlwaysRecording)),
		),
		fmt.Sprintf("%s: %v",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Automatic Scroll"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%v", m.automaticScroll)),
		),
		fmt.Sprintf("%s: %v",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Speaking"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%v", m.speaking)),
		),
	}, "\n"),
	)

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
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go orchestrator.ListenForSpeech(ctx, orchestration.Callbacks{
		OnTranscription: func(transcript string) {
			program.Send(stdoutMsg(transcript))
		},
		OnInterimTranscription: func(transcript string) {
			program.Send(interimTranscriptMsg(transcript))
		},
		OnSpeakingStateChanged: func(isSpeaking bool) {
			program.Send(speakingMsg(isSpeaking))
		},
	})

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
