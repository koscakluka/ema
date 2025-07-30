package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/koscakluka/ema/core/llms/groq"
	"github.com/koscakluka/ema/core/speechtotext"
	deepgrams2t "github.com/koscakluka/ema/core/speechtotext/deepgram"
	"github.com/koscakluka/ema/core/texttospeech"
	deepgramt2s "github.com/koscakluka/ema/core/texttospeech/deepgram"

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

	bufferSize = 128
)

type stdoutMsg string
type speakingMsg bool

type interimTranscriptMsg string
type endRecordingMsg struct{}

var program *tea.Program

var isRecording bool
var output strings.Builder
var mutex sync.RWMutex

type model struct {
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
			isRecording = true
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

		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case endRecordingMsg:
		isRecording = false
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
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%v", isRecording)),
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
	program = tea.NewProgram(
		model{automaticScroll: true, ready: false},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go listenForSpeech(ctx)

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

	in := make([]int16, bufferSize)
	out := make([]int16, bufferSize)
	stream, err := portaudio.OpenDefaultStream(1, 1, 48000, bufferSize, in, out)
	if err != nil {
		log.Fatalf("Failed to open PortAudio stream: %v", err)
	}
	defer stream.Close()

	voice := deepgramt2s.VoiceAuraAsteria
	fmt.Println("Using voice", voice)
	voiceInfo := deepgramt2s.GetVoiceInfo(voice)
	fmt.Println("Gender:", voiceInfo.Age, voiceInfo.Gender)
	fmt.Println("Language:", voiceInfo.Language)
	fmt.Println("Characteristics:", voiceInfo.Characteristics)
	fmt.Println("UseCases:", voiceInfo.UseCases)
	deepgramSpeechClient, err := deepgramt2s.NewTextToSpeechClient(context.TODO(), voice)
	if err != nil {
		fmt.Printf("Failed to create deepgram speech client: %v", err)
	}

	leftoverAudio := make([]byte, bufferSize*2)

	if err := deepgramSpeechClient.OpenStream(context.TODO(),
		texttospeech.WithAudioCallback(func(audio []byte) {
			bufferSize := bufferSize * 2

			// PERF: This is just to test this, there is no reason we should
			// kill performance by copying here
			audio = append(leftoverAudio, audio...)
			for i := range len(audio)/bufferSize + 1 {
				if (i+1)*bufferSize > len(audio) {
					leftoverAudio = make([]byte, len(audio)-i*bufferSize)
					copy(leftoverAudio, audio[i*bufferSize:])
					break
				}

				binary.Read(bytes.NewBuffer(audio[i*bufferSize:(i+1)*bufferSize]), binary.LittleEndian, out)
				stream.Write()
			}
		}),
		texttospeech.WithAudioEndedCallback(func(transcript string) {
			bufferSize := bufferSize * 2
			lastAudio := make([]byte, bufferSize)
			for i := range bufferSize {
				lastAudio[i] = 0
			}
			copy(lastAudio, leftoverAudio)
			binary.Write(bytes.NewBuffer(lastAudio), binary.LittleEndian, out)
			stream.Write()
			return
		}),
	); err != nil {
		fmt.Printf("Failed to open deepgram speech stream: %v", err)
	}

	deepgramClient := deepgrams2t.NewClient(context.TODO())
	if err = deepgramClient.Transcribe(context.TODO(),
		speechtotext.WithSpeechStartedCallback(func() { program.Send(speakingMsg(true)) }),
		speechtotext.WithSpeechEndedCallback(func() { program.Send(speakingMsg(false)) }),
		speechtotext.WithInterimTranscriptionCallback(func(transcript string) { program.Send(interimTranscriptMsg(transcript)) }),
		speechtotext.WithTranscriptionCallback(func(transcript string) {
			program.Send(interimTranscriptMsg(""))
			program.Send(stdoutMsg(transcript + "\n"))
			flushedOnFinal := false
			client.Prompt(context.TODO(), transcript, func(data string) {
				flushedOnFinal = false
				fmt.Print(data)
				if err := deepgramSpeechClient.SendText(data); err != nil {
					log.Printf("Failed to send text to deepgram: %v", err)
				}
			})
			if !flushedOnFinal {
				if err := deepgramSpeechClient.FlushBuffer(); err != nil {
					log.Printf("Failed to flush buffer: %v", err)
				}
			}
			fmt.Println()
		}),
	); err != nil {
		log.Fatalf("Failed to start transcribing: %v", err)
	}
	defer deepgramClient.Close()

	if err := stream.Start(); err != nil {
		log.Fatalf("Failed to start PortAudio stream: %v", err)
	}
	fmt.Println("Starting microphone capture. Speak now...")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := stream.Read(); err != nil {
				log.Printf("Failed to read from PortAudio stream: %v", err)
			}
			if isRecording {
				audioBuffer := bytes.Buffer{}
				binary.Write(&audioBuffer, binary.LittleEndian, in)
				if err := deepgramClient.SendAudio(audioBuffer.Bytes()); err != nil {
					log.Fatalf("Failed to send audio: %v", err)
				}
			}
		}
	}

}
