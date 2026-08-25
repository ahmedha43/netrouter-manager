package network

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

var (
	logMu       sync.RWMutex
	logRingBuf  []protocol.LogEntry
	maxLogCount = 200
)

func AddLogEntry(facility, message string) {
	logMu.Lock()
	defer logMu.Unlock()

	entry := protocol.LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Facility:  facility,
		Message:   message,
	}

	if len(logRingBuf) >= maxLogCount {
		logRingBuf = logRingBuf[1:]
	}
	logRingBuf = append(logRingBuf, entry)
}

func ReadSystemLogs() protocol.SystemLogs {
	logMu.RLock()
	defer logMu.RUnlock()

	var entries []protocol.LogEntry
	// Include in-memory ring buffer
	entries = append(entries, logRingBuf...)

	// Also read /var/log/messages if present
	if file, err := os.Open("/var/log/messages"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				entries = append(entries, protocol.LogEntry{
					Timestamp: time.Now().Format("2006-01-02 15:04:05"),
					Facility:  "SYSTEM",
					Message:   line,
				})
			}
		}
	}

	return protocol.SystemLogs{Entries: entries}
}
