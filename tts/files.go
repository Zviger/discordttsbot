package tts

import (
	"fmt"
	"io"
	"net/http"
	"os"
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

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
