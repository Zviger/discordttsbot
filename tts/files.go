package tts

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (s *Service) generateVoiceFilePath(fileName string) string {
	return fmt.Sprintf("./tmp/assets/input_%s.wav", fileName)
}

func (s *Service) DeleteVoiceFile(fileName string) error {
	filePath := s.generateVoiceFilePath(fileName)

	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ListVoiceFiles() ([]string, error) {
	files, err := filepath.Glob("./tmp/assets/input_*.wav")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, file := range files {
		names = append(names, strings.TrimSuffix(
			strings.TrimPrefix(filepath.Base(file), "input_"),
			".wav",
		))
	}

	return names, nil
}

func (s *Service) UploadVoiceFile(fileUrl, fileName string) error {
	filePath := s.generateVoiceFilePath(fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(fileUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 3. Create FFmpeg command that handles everything
	finalPath := s.generateVoiceFilePath(fileName)
	cmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0", // Read from stdin
		"-t", "60",
		// "-ac", "2", // Mono
		// "-ar", "24000", // Sample rate
		// "-acodec", "pcm_s16le", // 16-bit PCM
		"-y",      // Overwrite if exists
		finalPath, // FFmpeg writes directly here
	)

	// Stream download directly to FFmpeg
	cmd.Stdin = resp.Body

	// Capture FFmpeg errors
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 4. Run the pipeline
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg processing failed: %v\n%s", err, stderr.String())
	}

	return nil
}
