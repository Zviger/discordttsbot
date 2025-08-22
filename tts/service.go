package tts

import (
	"bytes"
	"discordttsbot/config"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/hraban/opus.v2"
)

type Service struct {
	config *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg}
}

func (s *Service) GenerateSpeechInWAV(text, voiceFileName, language string) ([]byte, error) {
	url := fmt.Sprintf(
		"%s/api/tts?text=%s&speaker_wav=%s&language_id=%s",
		s.config.TTS.ApiUrl,
		text,
		fmt.Sprintf("/root/assets/input_%s.wav", voiceFileName),
		language,
	)

	// Download the WAV file
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the WAV file into memory
	wavData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return wavData, nil
}

func (s *Service) convertWAVToOpusFrames(wavData []byte, volume float64) ([][]byte, error) {
	// Convert WAV to PCM (simplified - you may need proper WAV parsing)
	// This assumes the WAV is 16-bit 24kHz mono
	pcmData := bytes.NewBuffer(wavData[44:]) // Skip WAV header (44 bytes)

	// Create Opus encoder
	encoder, err := opus.NewEncoder(24000, 1, opus.AppAudio)
	if err != nil {
		return nil, err
	}

	frameSize := 480 // 20ms frame at 24kHz
	pcm := make([]int16, frameSize)
	adjustedPcm := make([]int16, frameSize)

	buffer := make([][]byte, 0)
	for {
		// Read PCM data
		err := binary.Read(pcmData, binary.LittleEndian, &pcm)
		if err == io.EOF {
			break
		} else if err != nil {
			break
		}

		for i, sample := range pcm {
			adjusted := float64(sample) * volume

			if adjusted > 32767 {
				adjusted = 32767
			} else if adjusted < -32768 {
				adjusted = -32768
			}

			adjustedPcm[i] = int16(adjusted)
		}

		// Encode to Opus
		opusFrame := make([]byte, 1000) // Sufficient size for Opus frame
		n, err := encoder.Encode(adjustedPcm, opusFrame)
		if err != nil {
			break
		}

		buffer = append(buffer, opusFrame[:n])
	}

	return buffer, nil
}

func (s *Service) GenerateSpeechInOpusFrames(text, voiceFileName, language string, volume float64) ([][]byte, error) {
	if volume < 0 {
		volume = 0
	} else if volume > 2.0 {
		volume = 2.0
	}

	wavData, err := s.GenerateSpeechInWAV(text, voiceFileName, language)
	if err != nil {
		return nil, err
	}

	opusFrames, err := s.convertWAVToOpusFrames(wavData, volume)
	if err != nil {
		return nil, err
	}

	return opusFrames, nil
}
