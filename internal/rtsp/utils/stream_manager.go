package utils

import (
	"github.com/pion/webrtc/v4/pkg/media/h264writer"
	"log"
	"os"
	"os/exec"
)

type Stream struct {
	Publisher string `json:"publisher"`
	URL       string `json:"url"`
}

type StreamManager struct {
	Idx   int
	file1 string
	file2 string
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		Idx:   1,
		file1: "test_first",
		file2: "test_second",
	}
}

func (sm *StreamManager) CreateVideo(file string) error {
	cmd := exec.Command("ffmpeg", "-i", file+".h264", "-c", "copy", "-c:v", "libx265", file+".mp4")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Println("Error converting H.264 to MP4:", err)
		return err
	}

	return nil
}

func (sm *StreamManager) CreateSuperResVideo(file string, model string, scale string) error {
	cmd := exec.Command("python", "superres.py", file, model, scale)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Println("Error during Super Res video:", err)
		return err
	}

	return nil
}

func (sm *StreamManager) PrepareWriter(file string) (*h264writer.H264Writer, error) {
	sm.Idx++

	w, err := h264writer.New(file + ".h264")
	if err != nil {
		return nil, err
	}
	return w, nil
}
