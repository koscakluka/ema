package deepgram

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/gorilla/websocket"
)

type TextToSpeechClient struct {
	wsConn           *websocket.Conn
	transcriptBuffer []string

	voice deepgramVoice
	mu    sync.Mutex
}

func NewTextToSpeechClient(ctx context.Context, voice deepgramVoice) (*TextToSpeechClient, error) {
	client := &TextToSpeechClient{voice: defaultVoice}

	if !slices.Contains(GetAvailableVoices(), voice) {
		return nil, fmt.Errorf("invalid voice")
	}

	client.voice = voice

	return client, nil
}

func (c *TextToSpeechClient) Close(ctx context.Context) {
	c.CloseStream(ctx)
}
