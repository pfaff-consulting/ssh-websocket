package failban

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type failLog struct {
	IP           string      `json:"ip"`
	Attempts     []time.Time `json:"attempts"`
	BlockedUntil time.Time   `json:"blocked_until"`
}

type Manager struct {
	config    *config
	failLogs  map[string]*failLog
	failMutex sync.RWMutex
}

type config struct {
	StorageFile string
	Attempts    int
	Timespan    int
	BanTime     int
}

func New(storageFile string, allowedAttempts int, timespan int, banTime int) *Manager {
	m := &Manager{
		config: &config{
			StorageFile: storageFile,
			Attempts:    allowedAttempts,
			Timespan:    timespan,
			BanTime:     banTime,
		},
	}
	m.loadFailLogs()
	return m
}

func (m *Manager) IsBlocked(ip string) bool {
	m.failMutex.RLock()
	failLog, exists := m.failLogs[ip]
	m.failMutex.RUnlock()
	if !exists {
		return false
	}
	return time.Now().Before(failLog.BlockedUntil)
}

func (m *Manager) AddFailedAttempt(ip string) {
	now := time.Now()
	m.failMutex.Lock()
	if m.failLogs[ip] == nil {
		m.failLogs[ip] = &failLog{IP: ip}
	}
	failLog := m.failLogs[ip]
	failLog.Attempts = append(failLog.Attempts, now)
	// Clean old attempts older than Timespan sec
	cutoff := now.Add(-time.Duration(m.config.Timespan) * time.Second)
	var newAttempts []time.Time
	for _, t := range failLog.Attempts {
		if t.After(cutoff) {
			newAttempts = append(newAttempts, t)
		}
	}
	failLog.Attempts = newAttempts
	if len(failLog.Attempts) > m.config.Attempts {
		failLog.BlockedUntil = now.Add(time.Duration(m.config.BanTime) * time.Second)
	}
	m.failMutex.Unlock()
	m.saveFailLogs()
}

func (m *Manager) loadFailLogs() {
	data, err := os.ReadFile(m.config.StorageFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading fail log: %v", err)
		}
		return
	}
	var logs map[string]*failLog
	if err := json.Unmarshal(data, &logs); err != nil {
		log.Printf("Error parsing fail log: %v", err)
		return
	}
	m.failMutex.Lock()
	m.failLogs = logs
	m.failMutex.Unlock()
}

func (m *Manager) saveFailLogs() {
	m.failMutex.RLock()
	data, err := json.MarshalIndent(m.failLogs, "", "  ")
	m.failMutex.RUnlock()
	if err != nil {
		log.Printf("Error marshaling fail logs: %v", err)
		return
	}
	if err := os.WriteFile(m.config.StorageFile, data, 0644); err != nil {
		log.Printf("Error writing fail log: %v", err)
	}
}
