package texttospeech

import "github.com/koscakluka/ema/core/audio"

type TextToSpeechOptions struct {
	AudioCallback func(audio []byte)
	AudioEnded    func(transcript string)

	EncodingInfo audio.EncodingInfo
}

type TextToSpeechOption func(*TextToSpeechOptions)

func WithAudioCallback(callback func(audio []byte)) TextToSpeechOption {
	return func(o *TextToSpeechOptions) {
		o.AudioCallback = callback
	}
}

func WithAudioEndedCallback(callback func(transcript string)) TextToSpeechOption {
	return func(o *TextToSpeechOptions) {
		o.AudioEnded = callback
	}
}

func WithEncodingInfo(encodingInfo audio.EncodingInfo) TextToSpeechOption {
	return func(o *TextToSpeechOptions) {
		if encodingInfo.SampleRate == 0 || encodingInfo.Encoding == "" {
			// TODO: Issue warning
			return
		}

		o.EncodingInfo = encodingInfo
	}
}
