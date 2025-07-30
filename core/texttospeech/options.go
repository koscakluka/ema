package texttospeech

type TextToSpeechOptions struct {
	AudioCallback func(audio []byte)
	AudioEnded    func(transcript string)
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
