package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type stdoutMsg string

var program *tea.Program
var output strings.Builder
var mutex sync.RWMutex

type model struct {
	termWidth  int
	termHeight int
	ready      bool

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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-5)
			m.viewport.SetContent(output.String())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case stdoutMsg:
		mutex.Lock()
		output.WriteString(string(msg))
		m.viewport.SetContent(output.String())
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
		Width(m.termWidth - 35).
		Height(m.termHeight - 3)

	sidebarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(33).
		Height(m.termHeight - 2)

	mainContent := mainStyle.Render(m.viewport.View())

	sidebar := sidebarStyle.Render(fmt.Sprintf("%s: %v\n",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("Automatic Scroll"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%v", m.automaticScroll)),
	))

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

	go func() {
		time.Sleep(2 * time.Second) // Give the UI time to start
		for i := range 1000 {
			fmt.Printf("Log message %d: This is some output to test the application\n", i)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
