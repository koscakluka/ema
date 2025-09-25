package deepgram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/koscakluka/ema/core/audio"
	"github.com/koscakluka/ema/core/texttospeech"
)

const (
	defaultSampleRate = 48000
	defaultEncoding   = "linear16"
)

func (c *TextToSpeechClient) OpenStream(ctx context.Context, opts ...texttospeech.TextToSpeechOption) error {
	options := texttospeech.TextToSpeechOptions{
		EncodingInfo: audio.EncodingInfo{
			SampleRate: defaultSampleRate,
			Encoding:   defaultEncoding,
		},
	}
	for _, opt := range opts {
		opt(&options)
	}

	conn, err := connectWebsocket(c.voice, options.EncodingInfo)
	if err != nil {
		return fmt.Errorf("failed to open websocket: %w", err)
	}

	c.wsConn = conn

	go c.readAndProcessMessages(ctx, conn, options)

	return nil
}

func connectWebsocket(voice deepgramVoice, encodingInfo audio.EncodingInfo) (*websocket.Conn, error) {
	apiKey, ok := os.LookupEnv("DEEPGRAM_API_KEY")
	if !ok {
		return nil, fmt.Errorf("deepgram api key not found")
	}

	urlValues := url.Values{}
	urlValues.Set("encoding", encodingInfo.Encoding)
	urlValues.Set("sample_rate", strconv.Itoa(encodingInfo.SampleRate))
	urlValues.Set("model", string(voice))
	urlValues.Set("container", "none")

	conn, _, err := websocket.DefaultDialer.Dial(
		(&url.URL{
			Scheme: "wss",
			Host:   "api.deepgram.com", Path: "/v1/speak",
			RawQuery: urlValues.Encode(),
		}).String(),
		http.Header{"Authorization": {"token " + apiKey}})
	if err != nil {
		return nil, fmt.Errorf("failed to open socket connection to deepgram: %w", err)
	}

	return conn, err
}

func (c *TextToSpeechClient) SendText(text string) error {
	if c.wsConn == nil {
		return fmt.Errorf("connection closed")
	}

	if err := c.wsConn.WriteJSON(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "Speak",
		Text: text,
	}); err != nil {
		return fmt.Errorf("failed to send text to deepgram through websocket: %w", err)
	}

	c.transcript += text
	return nil
}

func (c *TextToSpeechClient) FlushBuffer() error {
	if c.wsConn == nil {
		return fmt.Errorf("connection closed")
	}
	if err := c.wsConn.WriteJSON(struct {
		Type string `json:"type"`
	}{
		Type: "Flush",
	}); err != nil {
		return fmt.Errorf("failed to flush deepgram buffer through websocket: %w", err)
	}

	return nil
}

func (c *TextToSpeechClient) ClearBuffer() error {
	if c.wsConn == nil {
		return fmt.Errorf("connection closed")
	}
	if err := c.wsConn.WriteJSON(struct {
		Type string `json:"type"`
	}{
		Type: "Clear",
	}); err != nil {
		return fmt.Errorf("failed to clear deepgram buffer through websocket: %w", err)
	}
	c.transcript = ""

	return nil
}

func (c *TextToSpeechClient) CloseStream(ctx context.Context) error {
	if c.wsConn != nil {
		if err := c.wsConn.WriteJSON(struct {
			Type string `json:"type"`
		}{
			Type: "Close",
		}); err != nil {
			log.Printf("Failed to send close message to deepgram websocket: %v", err)
		}

	}

	return nil
}

func (c *TextToSpeechClient) readAndProcessMessages(_ context.Context, conn *websocket.Conn, options texttospeech.TextToSpeechOptions) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if err.Error() != "websocket: close 1000 (normal)" {
				log.Printf("Websocket read error: %v", err)
			}

			c.wsConn.Close()
			c.wsConn = nil

			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			if options.AudioCallback != nil && len(msg) > 0 {
				options.AudioCallback(msg)
			}
		default:
			// c.logger.Debug("deepgram message", slog.String("msg", string(msg)))
			var parsedMsg struct {
				Type string `json:"type"`
			}
			err := json.Unmarshal(msg, &parsedMsg)
			if err != nil {
				log.Printf("Failed to unmarshal deepgram message: %v", err)
				continue
			}

			switch parsedMsg.Type {
			case "Flushed":
				if options.AudioEnded != nil {
					options.AudioEnded(c.transcript)
				}
				c.transcript = ""
			}
		}
	}
}
