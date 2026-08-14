package handler

import (
	"fmt"
	"net/http"
	"strings"
)

type SSEWriter struct {
	w http.ResponseWriter
}

func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	return &SSEWriter{w: w}
}

func (s *SSEWriter) WriteEvent(event string, data string) {
	s.w.Write([]byte(fmt.Sprintf("event: %s\n", event)))
	s.w.Write([]byte(fmt.Sprintf("data: %s\n\n", strings.TrimSpace(data))))
	s.w.(http.Flusher).Flush()
}

func (s *SSEWriter) WriteData(data string) {
	s.w.Write([]byte(fmt.Sprintf("data: %s\n\n", strings.TrimSpace(data))))
	s.w.(http.Flusher).Flush()
}

func (s *SSEWriter) Done(data string) {
	s.w.Write([]byte("event: done\n"))
	s.w.Write([]byte(fmt.Sprintf("data: %s\n\n", strings.TrimSpace(data))))
	s.w.(http.Flusher).Flush()
}
