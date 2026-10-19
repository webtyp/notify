//go:build !wasm

package notify_test

import (
	"testing"
	"webtyp.com/notify"
)

func TestAlert_Validate(t *testing.T) {
	tests := []struct {
		name  string
		alert notify.Alert
		want  error
	}{
		{
			name:  "valid",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World"},
			want:  nil,
		},
		{
			name:  "empty code",
			alert: notify.Alert{Subject: "Hello", Body: "World"},
			want:  notify.ErrAlertCode,
		},
		{
			name:  "empty subject",
			alert: notify.Alert{Code: "test", Body: "World"},
			want:  notify.ErrAlertText,
		},
		{
			name:  "empty body",
			alert: notify.Alert{Code: "test", Subject: "Hello"},
			want:  notify.ErrAlertText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.alert.Validate()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
