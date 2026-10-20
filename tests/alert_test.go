//go:build !wasm

package notify_test

import (
	"testing"

	"webtyp.com/msgtype"
	"webtyp.com/notify"
)

func TestAlert_Validate(t *testing.T) {
	tests := []struct {
		name  string
		alert notify.Alert
		want  error
	}{
		{
			name:  "valid (Info)",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Info},
			want:  nil,
		},
		{
			name:  "valid with source",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Info, Source: notify.Ref{Type: "item", ID: "123"}},
			want:  nil,
		},
		{
			name:  "valid (Warning)",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Warning},
			want:  nil,
		},
		{
			name:  "valid (Error)",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Error},
			want:  nil,
		},
		{
			name:  "invalid level (Normal)",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Normal},
			want:  notify.ErrAlertLevel,
		},
		{
			name:  "invalid level (Success)",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Success},
			want:  notify.ErrAlertLevel,
		},
		{
			name:  "invalid level (Debug)",
			alert: notify.Alert{Code: "test", Subject: "Hello", Body: "World", Level: msgtype.Debug},
			want:  notify.ErrAlertLevel,
		},
		{
			name:  "empty code",
			alert: notify.Alert{Subject: "Hello", Body: "World", Level: msgtype.Info},
			want:  notify.ErrAlertCode,
		},
		{
			name:  "empty subject",
			alert: notify.Alert{Code: "test", Body: "World", Level: msgtype.Info},
			want:  notify.ErrAlertText,
		},
		{
			name:  "empty body",
			alert: notify.Alert{Code: "test", Subject: "Hello", Level: msgtype.Info},
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
