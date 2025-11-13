package miniaudio

import (
	"fmt"
	"sync"

	"github.com/gen2brain/malgo"
)

type playbackClient struct {
	audioContext *malgo.AllocatedContext
	device       *malgo.Device
	config       malgo.DeviceConfig

	leftoverAudio []byte
	marks         []playbackMark

	mu      sync.Mutex
	audioMu sync.Mutex
	marksMu sync.Mutex
}

func (c *playbackClient) Init(audioContext *malgo.AllocatedContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	sampleRate := uint32(sampleRate)
	channels := 1
	format := malgo.FormatS16
	bytesPerFrame := malgo.SampleSizeInBytes(format) * channels

	c.config = malgo.DefaultDeviceConfig(malgo.Playback)
	c.config.SampleRate = sampleRate
	c.config.Playback.Format = format
	c.config.Playback.Channels = uint32(channels)
	c.config.Alsa.NoMMap = 1
	c.config.PeriodSizeInFrames = 480 // ~10ms at 48kHz
	c.config.Periods = 4

	c.audioContext = audioContext

	var err error
	if c.device, err = malgo.InitDevice(c.audioContext.Context, c.config, malgo.DeviceCallbacks{
		Data: func(pOutput, _ []byte, frameCount uint32) {
			need := int(frameCount) * bytesPerFrame
			written := 0

			for written < need {
				var cur []byte = nil
				if len(c.leftoverAudio) > 0 {
					passedMarks := 0
					for i, mark := range c.marks {
						if mark.position >= need {
							c.marks[i].position -= need
						} else {
							passedMarks++
						}
					}
					if passedMarks > 0 {
						go func() {
							c.marksMu.Lock()
							defer c.marksMu.Unlock()
							toCall := c.marks[:passedMarks]
							c.marks = c.marks[passedMarks:]
							for _, mark := range toCall {
								mark.callback(mark.name)
							}
						}()
					}
					if len(c.leftoverAudio) < need {
						cur = c.leftoverAudio
						c.audioMu.Lock()
						c.leftoverAudio = make([]byte, 0)
						c.audioMu.Unlock()
					} else {
						cur = c.leftoverAudio[:need]
						c.audioMu.Lock()
						c.leftoverAudio = c.leftoverAudio[need:]
						c.audioMu.Unlock()
					}
				}
				if cur == nil {
					for i := written; i < need; i++ {
						pOutput[i] = 0
					}
					break
				}
				n := copy(pOutput[written:need], cur)
				written += n
			}
		},
	}); err != nil {
		return err
	}

	return nil
}

func (c *playbackClient) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.device == nil {
		return fmt.Errorf("device not initialized")
	}

	if err := c.device.Start(); err != nil {
		return fmt.Errorf("failed to start playback device: %w", err)
	}

	return nil
}

func (c *playbackClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.device == nil {
		return fmt.Errorf("device not initialized")
	}

	if err := c.device.Stop(); err != nil {
		return fmt.Errorf("failed to stop playback device: %w", err)
	}

	c.ClearBuffer()
	return nil
}

func (c *playbackClient) SendAudio(audio []byte) error {
	if c.device == nil {
		return fmt.Errorf("device not initialized")
	} else if !c.device.IsStarted() {
		return fmt.Errorf("device not started")
	}

	c.audioMu.Lock()
	defer c.audioMu.Unlock()
	c.leftoverAudio = append(c.leftoverAudio, audio...)
	return nil
}

func (c *playbackClient) ClearBuffer() {
	c.audioMu.Lock()
	c.marksMu.Lock()
	defer c.audioMu.Unlock()
	defer c.marksMu.Unlock()
	c.leftoverAudio = make([]byte, 0)
	c.marks = nil

}

func (c *playbackClient) AwaitMark() error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go c.Mark("", func(string) { wg.Done() })
	wg.Wait()
	return nil
}

func (c *playbackClient) Mark(mark string, callback func(string)) error {
	c.marksMu.Lock()
	defer c.marksMu.Unlock()
	c.marks = append(c.marks, playbackMark{
		name:     mark,
		position: len(c.leftoverAudio),
		callback: callback,
	})
	return nil
}

func (c *playbackClient) Uninit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.device == nil {
		return fmt.Errorf("device not initialized")
	}

	c.device.Uninit()
	c.device = nil

	return nil
}

type playbackMark struct {
	name     string
	position int
	callback func(string)
}
