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
	awaiting      bool
	wait          sync.WaitGroup

	mu      sync.Mutex
	audioMu sync.Mutex
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
					if c.awaiting {
						c.wait.Done()
						c.awaiting = false
					}

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
	defer c.audioMu.Unlock()
	c.leftoverAudio = make([]byte, 0)
}

func (c *playbackClient) AwaitMark() error {
	c.wait.Add(1)
	c.awaiting = true
	c.wait.Wait()
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
