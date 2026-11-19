//go:build !wasm

package notify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"webtyp.com/msgtype"
)

func TestCriticalFileLog_SkipsInfoAndWritesCritical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.log")
	fl := NewCriticalFileLog(path)

	if err := fl.WriteAlert(Alert{Code: "info.ok", Level: msgtype.Info, Subject: "ok", Body: "normal"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Info alert must not create logs.log")
	}

	if err := fl.WriteAlert(Alert{Code: "security.malicious_content_blocked", Scope: "security", Level: msgtype.Error, Subject: "CSP", Body: "script-src"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "ERROR security.malicious_content_blocked") || !strings.Contains(got, "body=script-src") {
		t.Fatalf("unexpected log content: %q", got)
	}
}

func TestCriticalFileLog_PrunesExpiredAndCapsSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.log")
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)

	oldLine := now.Add(-8 * 24 * time.Hour).Format(time.RFC3339) + " ERROR security.old body=stale\n"
	recentLine := now.Add(-2 * 24 * time.Hour).Format(time.RFC3339) + " WARNING security.recent body=keep\n"
	if err := os.WriteFile(path, []byte(oldLine+recentLine), 0600); err != nil {
		t.Fatal(err)
	}

	fl := NewCriticalFileLog(path)
	fl.now = func() time.Time { return now }

	if err := fl.WriteAlert(Alert{Code: "security.new", Level: msgtype.Error, Subject: "now", Body: "fresh"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Contains(got, "security.old") {
		t.Fatalf("expired entry (>7d) should be pruned, got:\n%s", got)
	}
	if !strings.Contains(got, "security.recent") || !strings.Contains(got, "security.new") {
		t.Fatalf("recent and new entries must remain, got:\n%s", got)
	}

	// Verify hard byte cap drops oldest entries.
	fl.maxBytes = 120
	for i := 0; i < 5; i++ {
		if err := fl.WriteAlert(Alert{Code: "security.burst", Level: msgtype.Error, Subject: "burst", Body: strings.Repeat("x", 40)}); err != nil {
			t.Fatal(err)
		}
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) > fl.maxBytes {
		t.Fatalf("logs.log size %d exceeds maxBytes %d", len(data), fl.maxBytes)
	}
}
