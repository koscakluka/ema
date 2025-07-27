package deepgram

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type TranscriptionClient struct {
	lastMsgTs time.Time

	accumulatedTranscript string
	unendedSegment        bool

	conn   *websocket.Conn
	connMu sync.Mutex
}

func NewClient(ctx context.Context) *TranscriptionClient {
	return &TranscriptionClient{}
}

func (s *TranscriptionClient) Close() error {
	return s.StopStream()
}
