package utils

import (
	"github.com/pion/webrtc/v4/pkg/media/h264writer"
	"log"
	"os"
	"os/exec"
	"strconv"
)

type Stream struct {
	Publisher string `json:"publisher"`
	URL       string `json:"url"`
}

type StreamManager struct {
	H264Writer  *h264writer.H264Writer
	Idx         int
	LastSrcFile string
	lastDstFile string
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		Idx:         1,
		LastSrcFile: "",
		lastDstFile: "",
	}
}

func (sm *StreamManager) CreateVideo() error {
	cmd := exec.Command("ffmpeg", "-i", sm.LastSrcFile, "-c", "copy", "-c:v", "libx265", sm.lastDstFile)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Println("Error converting H.264 to MP4:", err)
		return err
	}
	return nil
}

func (sm *StreamManager) PrepareWriter() (*h264writer.H264Writer, error) {
	sm.LastSrcFile = "input" + strconv.Itoa(sm.Idx) + ".h264"
	sm.lastDstFile = "output" + strconv.Itoa(sm.Idx) + ".mp4"
	sm.Idx++
	w, err := h264writer.New(sm.LastSrcFile)
	if err != nil {
		log.Fatalln("Fatal error during h264writer.New(filename): ", err)
	}
	return w, nil
}
