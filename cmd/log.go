package main

import (
	"container/ring"
	"log/slog"
	"net/http"
	"time"
)

// @Title
// @Description
// @Author
// @Update

// LogHistory keeps a circular log of the last N lines of logs for output via the /log
type LogHistory struct {
	History *ring.Ring
	Length  int
}

// NewLogHistory records a circular buffer of log lines of length size
func NewLogHistory(length int) *LogHistory {
	h := &LogHistory{
		History: ring.New(length),
		Length:  length,
	}
	return h
}

// Write so that this satisfies the Writer interface so that we can use this in a MultiWriter
func (h *LogHistory) Write(buf []byte) (n int, err error) {
	h.History.Value = string(buf)
	h.History = h.History.Next()
	return len(buf), nil
}

// String converts the full buffer to a single string output for display in a web page
func (h *LogHistory) Lines() []string {
	var lines []string
	h.History.Do(func(p interface{}) {
		if p != nil {
			lines = append(lines, string(p.(string)))
		}
	})
	return lines
}

// statusWriter records the status code and body size of a response for logRequests.
type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.size += n
	return n, err
}

// Unwrap lets http.ResponseController reach Flush and the deadlines underneath.
func (s *statusWriter) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// logRequests logs every request once it completes.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"duration_seconds", time.Since(start).Seconds(),
			"url", r.URL.String(),
			"path", r.URL.Path,
			"size", sw.size,
			"status", sw.status)
	})
}
