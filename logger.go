package main

import (
	"encoding/json"
	"sync"
)

type LogEntry struct {
	Type    string `json:"type"` // "reading", "flagged", "skipped", "error", "info"
	Message string `json:"message"`
}

type LogBroadcaster struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

var logBus = &LogBroadcaster{
	clients: make(map[chan string]struct{}),
}

func (b *LogBroadcaster) Subscribe() chan string {
	ch := make(chan string, 200)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *LogBroadcaster) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

func (b *LogBroadcaster) emit(entry LogEntry) {
	data, _ := json.Marshal(entry)
	msg := string(data)
	b.mu.Lock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
	b.mu.Unlock()
}

func LogInfo(message string) {
	logBus.emit(LogEntry{Type: "info", Message: message})
}

func LogReading(subject, from string) {
	logBus.emit(LogEntry{Type: "reading", Message: "Reading: " + subject + "  (" + from + ")"})
}

func LogFlagged(company, title, status string) {
	logBus.emit(LogEntry{Type: "flagged", Message: "✓  " + company + " — " + title + " [" + status + "]"})
}

func LogSkipped(subject string) {
	logBus.emit(LogEntry{Type: "skipped", Message: "✗  " + subject})
}

func LogError(message string) {
	logBus.emit(LogEntry{Type: "error", Message: message})
}
