package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"slices"

	"os"
	"strings"
	"sync"
	"time"

	"github.com/koscakluka/ema/core"
	"github.com/koscakluka/ema/core/audio/portaudio"
	"github.com/koscakluka/ema/core/llms/groq"
	deepgrams2t "github.com/koscakluka/ema/core/speechtotext/deepgram"
	deepgramt2s "github.com/koscakluka/ema/core/texttospeech/deepgram"
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

	speechBufferingDelay   = time.Millisecond * 50
	noSpeechBufferingDelay = time.Millisecond * 10

	bufferSize = 128
)

type speechDetectedMsg bool
type endRecordingMsg struct{}

type transcriptMsg string
type responseMsg string
type responseEndMsg struct{}
type bufferingTickMsg struct{}
type interimTranscriptMsg string
type cancelMsg struct{}

var program *tea.Program

var mutex sync.RWMutex

type promptPair struct {
	prompt   string
	response *promptResponse
}

type promptResponse struct {
	response       string
	displayedUntil int
	fullyReceived  bool
}

type model struct {
	output            *strings.Builder
	orchestrator      *orchestration.Orchestrator
	buffer            *bytes.Buffer
	buffering         *bool
	receivingResponse *bool

	termWidth         int
	termHeight        int
	ready             bool
	speechDetected    bool
	interimTranscript string

	viewport        viewport.Model
	automaticScroll bool

	endRecordingTimer *time.Timer
	promptsQueue      []promptPair // TODO: Consider using a ring-buffer or 2-way list if gc becomes an issue
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
		log.Println("Redirected stdout to log file")
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
			m.orchestrator.SetAlwaysRecording(!m.orchestrator.AlwaysRecording)

		case "m":
			m.orchestrator.SetSpeaking(!m.orchestrator.IsSpeaking)

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
		if m.automaticScroll {
			m.viewport.GotoBottom()
		}

	case speechDetectedMsg:
		m.speechDetected = bool(msg)

	case cancelMsg:
		*m.buffering = false
		var pair promptPair
		pair, m.promptsQueue = m.promptsQueue[0], m.promptsQueue[1:]

		mutex.Lock()
		m.output.WriteString(pair.prompt + "\n")
		if pair.response != nil && pair.response.displayedUntil > 0 {
			m.output.WriteString(pair.response.response[:pair.response.displayedUntil] + "\n\n")
		}
		mutex.Unlock()

	case bufferingTickMsg:
		if len(m.promptsQueue) == 0 {
			*m.buffering = false
			return m, nil
		}

		if m.promptsQueue[0].response != nil {
			if m.promptsQueue[0].response.displayedUntil < len(m.promptsQueue[0].response.response) {
				m.promptsQueue[0].response.displayedUntil++
			} else if m.promptsQueue[0].response.fullyReceived {
				var pair promptPair
				pair, m.promptsQueue = m.promptsQueue[0], m.promptsQueue[1:]

				mutex.Lock()
				m.output.WriteString(pair.prompt + "\n")
				m.output.WriteString(pair.response.response + "\n\n")
				mutex.Unlock()
			}

		}

		m.viewport.SetContent(m.getContent())
		if m.automaticScroll {
			m.viewport.GotoBottom()
		}

	case responseMsg:
		firstIncompleteResponse := slices.IndexFunc(m.promptsQueue, func(p promptPair) bool {
			if p.response == nil {
				return true
			}

			return !p.response.fullyReceived
		})

		if firstIncompleteResponse == -1 {
			return m, nil
		}

		if m.promptsQueue[firstIncompleteResponse].response == nil {
			m.promptsQueue[firstIncompleteResponse].response = &promptResponse{
				response:       string(msg),
				displayedUntil: 0,
				fullyReceived:  false,
			}
		} else {
			m.promptsQueue[firstIncompleteResponse].response.response += string(msg)
		}

		if !*m.buffering {
			*m.buffering = true
			return m, tea.Cmd(func() tea.Msg {
				for {
					if !*m.buffering {
						return nil
					}
					program.Send(bufferingTickMsg{})
					if m.orchestrator.IsSpeaking {
						time.Sleep(speechBufferingDelay)
					} else {
						time.Sleep(noSpeechBufferingDelay)
					}
				}
			})
		}
		return m, nil

	case responseEndMsg:
		firstIncompleteResponse := slices.IndexFunc(m.promptsQueue, func(p promptPair) bool {
			if p.response == nil {
				return true
			}

			return !p.response.fullyReceived
		})
		if firstIncompleteResponse == -1 {
			return m, nil
		}

		if m.promptsQueue[firstIncompleteResponse].response == nil {
			m.promptsQueue[firstIncompleteResponse].response = &promptResponse{
				response:       "",
				displayedUntil: 0,
				fullyReceived:  true,
			}
		} else {
			m.promptsQueue[firstIncompleteResponse].response.fullyReceived = true
		}

	case transcriptMsg:
		m.promptsQueue = append(m.promptsQueue, promptPair{prompt: string(msg)})

		m.viewport.SetContent(m.getContent())
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
	output := strings.TrimSpace(m.output.String())
	if len(output) > 0 {
		output += "\n\n"
	}
	for _, prompt := range m.promptsQueue {
		output += prompt.prompt + "\n"
		if prompt.response == nil || prompt.response.displayedUntil == 0 {
			// TODO: Add color
			output += "...\n\n"
		} else {
			output += prompt.response.response[:prompt.response.displayedUntil] + "\n\n"
		}
	}
	if m.interimTranscript != "" {
		output += strings.TrimSpace(m.interimTranscript)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deepgramClient := deepgrams2t.NewClient(ctx)
	defer deepgramClient.Close()

	voice := deepgramt2s.VoiceAuraAsteria
	deepgramSpeechClient, err := deepgramt2s.NewTextToSpeechClient(context.TODO(), voice)
	if err != nil {
		log.Printf("Failed to create deepgram speech client: %v", err)
	}
	defer deepgramSpeechClient.Close(ctx)

	audioClient, err := portaudio.NewClient(128)

	llm, err := groq.NewClient()
	if err != nil {
		log.Fatalf("Failed to create groq client: %v", err)
	}

	orchestrator := orchestration.NewOrchestrator(
		orchestration.WithLLM(llm),
		orchestration.WithSpeechToTextClient(deepgramClient),
		orchestration.WithTextToSpeechClient(deepgramSpeechClient),
		orchestration.WithAudioInput(audioClient),
		orchestration.WithAudioOutput(audioClient),
		orchestration.WithOrchestrationTools(),
	)

	program = tea.NewProgram(
		model{
			output:            &strings.Builder{},
			automaticScroll:   true,
			orchestrator:      orchestrator,
			buffer:            bytes.NewBuffer([]byte{}),
			buffering:         utils.Ptr(false),
			receivingResponse: utils.Ptr(false),
			promptsQueue:      []promptPair{},
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	orchestrator.Orchestrate(ctx,
		orchestration.WithTranscriptionCallback(func(transcript string) {
			program.Send(transcriptMsg(transcript))
		}),
		orchestration.WithInterimTranscriptionCallback(func(transcript string) {
			program.Send(interimTranscriptMsg(transcript))
		}),
		orchestration.WithSpeakingStateChangedCallback(func(isSpeaking bool) {
			program.Send(speechDetectedMsg(isSpeaking))
		}),
		orchestration.WithResponseCallback(func(response string) {
			program.Send(responseMsg(response))
		}),
		orchestration.WithResponseEndCallback(func() {
			program.Send(responseEndMsg{})
		}),
		orchestration.WithCancellationCallback(func() {
			program.Send(cancelMsg{})
			audioClient.ClearBuffer()
		}),
	)
	defer orchestrator.Close()

	if _, err := program.Run(); err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
