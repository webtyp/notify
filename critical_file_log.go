//go:build !wasm

package notify

import (
	"os"
	"strings"
	"sync"
	"time"

	"webtyp.com/msgtype"
)

const (
	// DefaultCriticalLogFile is the default path in the project root where
	// critical alerts are persisted so observability survives process restarts.
	DefaultCriticalLogFile = "logs.log"

	// DefaultCriticalLogMaxAge is the retention window for entries in logs.log.
	DefaultCriticalLogMaxAge = 7 * 24 * time.Hour

	// DefaultCriticalLogMaxBytes is the hard size cap for logs.log (1 MiB).
	DefaultCriticalLogMaxBytes = 1 << 20
)

// CriticalFileLog persists Warning and Error alerts as single ISO-8601 lines in
// a bounded local file, pruning entries older than MaxAge and capping total
// size at MaxBytes. Info alerts are ignored.
type CriticalFileLog struct {
	mu       sync.Mutex
	path     string
	maxAge   time.Duration
	maxBytes int64
	now      func() time.Time
}

// NewCriticalFileLog creates a critical alert log writer at path (defaults to
// DefaultCriticalLogFile when path is empty), retaining 7 days up to 1 MiB.
func NewCriticalFileLog(path string) *CriticalFileLog {
	if path == "" {
		path = DefaultCriticalLogFile
	}
	return &CriticalFileLog{
		path:     path,
		maxAge:   DefaultCriticalLogMaxAge,
		maxBytes: DefaultCriticalLogMaxBytes,
		now:      time.Now,
	}
}

// WriteAlert persists a when its level is Warning or Error, pruning expired
// lines and trimming oldest entries when the size cap is exceeded.
func (l *CriticalFileLog) WriteAlert(a Alert) error {
	if a.Level != msgtype.Warning && a.Level != msgtype.Error {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	cutoff := now.Add(-l.maxAge)

	var kept []string
	if data, err := os.ReadFile(l.path); err == nil && len(data) > 0 {
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			if ts, ok := parseLineTimestamp(line); ok && ts.Before(cutoff) {
				continue
			}
			kept = append(kept, line)
		}
	}

	kept = append(kept, formatAlertLine(now, a))
	kept = trimToMaxBytes(kept, l.maxBytes)

	content := strings.Join(kept, "\n") + "\n"
	return os.WriteFile(l.path, []byte(content), 0600)
}

func formatAlertLine(ts time.Time, a Alert) string {
	level := strings.ToUpper(a.Level.String())
	var b strings.Builder
	b.WriteString(ts.Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(level)
	b.WriteByte(' ')
	b.WriteString(sanitizeField(a.Code))
	if a.Scope != "" {
		b.WriteString(" scope=")
		b.WriteString(sanitizeField(a.Scope))
	}
	if a.Subject != "" {
		b.WriteString(" subject=")
		b.WriteString(sanitizeField(a.Subject))
	}
	if a.Body != "" {
		b.WriteString(" body=")
		b.WriteString(sanitizeField(a.Body))
	}
	return b.String()
}

func sanitizeField(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func parseLineTimestamp(line string) (time.Time, bool) {
	sp := strings.IndexByte(line, ' ')
	if sp <= 0 {
		return time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339, line[:sp])
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func trimToMaxBytes(lines []string, maxBytes int64) []string {
	if maxBytes <= 0 || len(lines) <= 1 {
		return lines
	}
	total := int64(0)
	for _, l := range lines {
		total += int64(len(l)) + 1
	}
	for len(lines) > 1 && total > maxBytes {
		total -= int64(len(lines[0])) + 1
		lines = lines[1:]
	}
	return lines
}
