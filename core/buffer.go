package orchestration

import (
	"sync"
)

// TODO: Optimize memory at some point, it is not a great idea to just append
// to a slice when we already consumed a part of it. But it needs to be synced
// properly, probably a ring buffer makes sense.

type buffer struct {
	chunks         []string
	chunksConsumed int
	chunksDone     bool
	chunksSignal   *sync.Cond

	audio           [][]byte
	audioConsumed   int
	audioDone       bool
	audioTranscript string
	audioSignal     *sync.Cond
}

func newBuffer() *buffer {
	return &buffer{
		chunksSignal: sync.NewCond(&sync.Mutex{}),
		audioSignal:  sync.NewCond(&sync.Mutex{}),
	}
}

func (b *buffer) AddChunk(chunk string) {
	b.chunks = append(b.chunks, chunk)
	b.chunksSignal.Broadcast()
}

func (b *buffer) ChunksDone() {
	b.chunksDone = true
	b.chunksSignal.Broadcast()
}

func (b *buffer) Chunks(yield func(string) bool) {
	for {
		b.chunksSignal.L.Lock()
		b.chunksSignal.Wait()
		b.chunksSignal.L.Unlock()
		for {
			if len(b.chunks) == b.chunksConsumed {
				break
			}
			chunk := b.chunks[b.chunksConsumed]
			b.chunksConsumed++
			if !yield(chunk) {
				return
			}
		}
		if b.chunksDone && b.chunksConsumed == len(b.chunks) {
			return
		}
	}
}

func (b *buffer) AddAudio(audio []byte) {
	b.audio = append(b.audio, audio)
	b.audioSignal.Broadcast()
}

func (b *buffer) AudioDone(transcript string) {
	b.audioDone = true
	b.audioTranscript = transcript
	b.audioSignal.Broadcast()
}

func (b *buffer) Audio(yield func(audio []byte) bool) {
	for {
		b.audioSignal.L.Lock()
		b.audioSignal.Wait()
		b.audioSignal.L.Unlock()
		for {
			if len(b.audio) == b.audioConsumed {
				break
			}
			audio := b.audio[b.audioConsumed]
			b.audioConsumed++
			if !yield(audio) {
				return
			}
		}
		if b.audioDone && b.audioConsumed == len(b.audio) {
			return
		}
	}
}

func (b *buffer) Clear() {
	// TODO: This should probably be locked
	b.chunks = []string{}
	b.chunksConsumed = 0
	b.chunksDone = true
	b.chunksSignal.Broadcast()
	b.chunksDone = false
	b.audio = [][]byte{}
	b.audioConsumed = 0
	b.audioDone = true
	b.audioSignal.Broadcast()
	b.audioDone = false
}
