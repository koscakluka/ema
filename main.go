package main

import (
	"context"
	"fmt"
	"io"

	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/koscakluka/ema/pkg/llms/groq"
	"github.com/koscakluka/ema/pkg/speechtotext"
	"github.com/koscakluka/ema/pkg/speechtotext/deepgram"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gordonklaus/portaudio"
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

var program *tea.Program

var output strings.Builder
var mutex sync.RWMutex

type model struct {
	termWidth  int
	termHeight int
	ready      bool
	speaking   bool

	viewport        viewport.Model
	automaticScroll bool
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
	viewportWidth := m.termWidth - sidebarOuterWidth - viewportPadding*2
	viewportHeight := m.termHeight - viewportPadding*2 - 3

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height

		viewportWidth = m.termWidth - sidebarOuterWidth - viewportPadding*2
		viewportHeight = m.termHeight - viewportPadding*2 - 3
		if !m.ready {
			m.viewport = viewport.New(viewportWidth, viewportHeight)
			m.ready = true
		} else {
			m.viewport.Width = viewportWidth
			m.viewport.Height = viewportHeight
		}
		m.viewport.SetContent(wordwrap.String(output.String(), viewportWidth-4))
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case speakingMsg:
		m.speaking = bool(msg)

	case stdoutMsg:
		mutex.Lock()
		output.WriteString(string(msg))
		m.viewport.SetContent(wordwrap.String(output.String(), viewportWidth-4))
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
	program = tea.NewProgram(
		model{automaticScroll: true, ready: false},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(2 * time.Second) // Give the UI time to start
		listenForSpeech(ctx)
	}()

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func listenForSpeech(ctx context.Context) {
	err := portaudio.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize PortAudio: %v", err)
	}
	defer portaudio.Terminate()

	client := groq.NewClient()

	deepgramClient := deepgram.NewClient(context.TODO())
	if err = deepgramClient.Transcribe(context.TODO(),
		speechtotext.WithSpeechStartedCallback(func() { program.Send(speakingMsg(true)) }),
		speechtotext.WithSpeechEndedCallback(func() { program.Send(speakingMsg(false)) }),
		speechtotext.WithPartialTranscriptionCallback(func(transcript string) { fmt.Println(transcript) }),
		speechtotext.WithTranscriptionCallback(func(transcript string) {
			client.Prompt(context.TODO(), transcript, func(data string) {
				fmt.Print(data)
			})
			fmt.Println()
		}),
	); err != nil {
		log.Fatalf("Failed to start transcribing: %v", err)
	}
	defer deepgramClient.Close()

	deviceInfo, err := portaudio.DefaultInputDevice()
	if err != nil {
		log.Fatalf("Failed to get default input device: %v", err)
	}
	fmt.Printf("Using device: %s\n", deviceInfo.Name)
	stream, err := portaudio.OpenDefaultStream(1, 0, 44000, 1024, func(in []int16) {
		audioBuffer := convertToBytes(in)
		if err := deepgramClient.SendAudio(audioBuffer); err != nil {
			log.Fatalf("Failed to send audio: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to open PortAudio stream: %v", err)
	}
	defer stream.Close()

	fmt.Println("Starting microphone capture. Speak now...")
	if err := stream.Start(); err != nil {
		log.Fatalf("Failed to start PortAudio stream: %v", err)
	}
	fmt.Println("Microphone stream started")
	<-ctx.Done()
}

func convertToBytes(audio []int16) []byte {
	audioBytes := make([]byte, len(audio)*2)
	for i, sample := range audio {
		audioBytes[i*2] = byte(sample & 0xff)
		audioBytes[i*2+1] = byte((sample >> 8) & 0xff)
	}
	return audioBytes
}
