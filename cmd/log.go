package main

import (
	"container/ring"
	"strings"
)

// @Title
// @Description
// @Author
// @Update

// LogHistory keeps a circular log of the last N lines of logs for output via the /log
type LogHistory struct {
	history *ring.Ring
	length  int
}

// NewLogHistory records a circular buffer of log lines of length size
func NewLogHistory(length int) *LogHistory {
	h := &LogHistory{
		history: ring.New(length),
		length:  length,
	}
	return h
}

func (h *LogHistory) Write(buf []byte) (n int, err error) {
	h.history.Value = string(buf)
	h.history = h.history.Next()
	return len(buf), nil
}

func (h *LogHistory) String() string {
	var lines []string
	h.history.Do(func(p interface{}) {
		if p != nil {
			lines = append(lines, string(p.(string)))
		}
	})
	return strings.Join(lines, "")
}
