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
func (h *LogHistory) String() string {
	var lines []string
	var b strings.Builder
	h.History.Do(func(p interface{}) {
		if p != nil {
			lines = append(lines, string(p.(string)))
		}
	})

	for i := len(lines); i > 0; i-- {
		b.WriteString(lines[i-1])
		b.WriteString("<br>")
	}
	return b.String()
}
