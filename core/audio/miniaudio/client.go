package miniaudio

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gen2brain/malgo"
	"github.com/koscakluka/ema/core/audio"
)

const sampleRate = 48000

type Client struct {
	audioContext *malgo.AllocatedContext

	pbDev  *malgo.Device
	capDev *malgo.Device

	leftoverAudio []byte

	awaiting bool
	wait     sync.WaitGroup
	mu       sync.Mutex
}

func NewClient() (*Client, error) {
	audioCtx, err := malgo.InitContext(
		nil,
		malgo.ContextConfig{},
		func(message string) { log.Println("malgo:", message) },
	)
	if err != nil {
		log.Fatalf("malgo InitContext failed: %v", err)
	}

	client := Client{
		audioContext: audioCtx,
	}

	sampleRate := uint32(sampleRate)
	channels := 1
	format := malgo.FormatS16
	bytesPerFrame := malgo.SampleSizeInBytes(format) * channels

	pbCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	pbCfg.SampleRate = sampleRate
	pbCfg.Playback.Format = format
	pbCfg.Playback.Channels = uint32(channels)
	pbCfg.Alsa.NoMMap = 1
	pbCfg.PeriodSizeInFrames = 480 // ~10ms at 48kHz
	pbCfg.Periods = 4

	client.pbDev, err = malgo.InitDevice(audioCtx.Context, pbCfg, malgo.DeviceCallbacks{
		Data: func(pOutput, _ []byte, frameCount uint32) {
			need := int(frameCount) * bytesPerFrame
			written := 0

			for written < need {
				var cur []byte = nil
				if len(client.leftoverAudio) > 0 {
					if len(client.leftoverAudio) < need {
						cur = client.leftoverAudio
						client.mu.Lock()
						client.leftoverAudio = make([]byte, 0)
						client.mu.Unlock()
					} else {
						cur = client.leftoverAudio[:need]
						client.mu.Lock()
						client.leftoverAudio = client.leftoverAudio[need:]
						client.mu.Unlock()
					}
				}
				if cur == nil {
					if client.awaiting {
						client.wait.Done()
						client.awaiting = false
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
	})
	if err != nil {
		_ = audioCtx.Uninit()
		audioCtx.Free()
		log.Fatalf("Init playback device failed: %v", err)
		return nil, err
	}

	if err := client.pbDev.Start(); err != nil {
		log.Fatalf("Start playback failed: %v", err)
	}

	return &client, nil
}

func (c *Client) Stream(ctx context.Context, onAudio func(audio []byte)) error {
	sampleRate := uint32(sampleRate)
	channels := 1
	format := malgo.FormatS16
	bytesPerFrame := malgo.SampleSizeInBytes(format) * channels

	capCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	capCfg.SampleRate = sampleRate
	capCfg.Capture.Format = format
	capCfg.Capture.Channels = uint32(channels)
	capCfg.Alsa.NoMMap = 1
	capCfg.PerformanceProfile = malgo.LowLatency
	capCfg.PeriodSizeInFrames = 480
	capCfg.Periods = 3

	var err error
	c.capDev, err = malgo.InitDevice(c.audioContext.Context, capCfg, malgo.DeviceCallbacks{
		Data: func(_, pInput []byte, frameCount uint32) {
			n := int(frameCount) * bytesPerFrame
			if len(pInput) < n || n == 0 {
				return
			}
			onAudio(pInput[:n])
		},
	})
	if err != nil {
		log.Fatalf("Init capture device failed: %v", err)
		return err
	}

	if err := c.capDev.Start(); err != nil {
		log.Fatalf("Start capture failed: %v", err)
	}

	return nil
}

func (c *Client) Close() {
	if c.capDev != nil {
		c.capDev.Uninit()
	}
	c.pbDev.Uninit()
	_ = c.audioContext.Uninit()
	c.audioContext.Free()
}

func (c *Client) SendAudio(audio []byte) error {
	if c.pbDev == nil {
		return fmt.Errorf("playback device not initialized")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.leftoverAudio = append(c.leftoverAudio, audio...)
	return nil
}

func (c *Client) ClearBuffer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.leftoverAudio = make([]byte, 0)
}

func (c *Client) AwaitMark() error {
	c.wait.Add(1)
	c.awaiting = true
	c.wait.Wait()
	return nil
}

func (c *Client) EncodingInfo() audio.EncodingInfo {
	return audio.EncodingInfo{
		SampleRate: sampleRate,
		Encoding:   "linear16",
	}
}
