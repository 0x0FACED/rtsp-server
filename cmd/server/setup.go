package server

import (
	"rtsp-server/internal/http"
	"rtsp-server/internal/rtsp"
)

func Execute() {
	media := rtsp.Setup()
	s := http.NewServer(media)
	s.Initialize()
}
