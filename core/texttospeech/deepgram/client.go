package deepgram

import (
	"context"
	"fmt"
	"slices"

	"github.com/gorilla/websocket"
)

type TextToSpeechClient struct {
	wsConn     *websocket.Conn
	transcript string

	voice deepgramVoice
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
