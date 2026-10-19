//go:build !wasm

package notify_test

import (
	"testing"

	"webtyp.com/events"
	"webtyp.com/events/mock"
	"webtyp.com/model"
	"webtyp.com/msgtype"
	"webtyp.com/notify"
)

type recordingFieldWriter struct {
	fields map[string]any
}

func (w *recordingFieldWriter) String(k, v string) { w.fields[k] = v }
func (w *recordingFieldWriter) Int(k string, v int64) { w.fields[k] = v }

// Implement dummy methods for other interface requirements of model.FieldWriter
func (w *recordingFieldWriter) Float(k string, v float64) {}
func (w *recordingFieldWriter) Bool(k string, v bool) {}
func (w *recordingFieldWriter) Bytes(k string, v []byte) {}
func (w *recordingFieldWriter) Null(k string) {}
func (w *recordingFieldWriter) Raw(k, v string) {}
func (w *recordingFieldWriter) Object(k string, v model.Encodable) {}
func (w *recordingFieldWriter) Array(k string, n int) model.ArrayWriter { return nil }

func TestAlert_IsNil(t *testing.T) {
	var a *notify.Alert
	if !a.IsNil() {
		t.Error("expected true for nil alert")
	}

	b := &notify.Alert{}
	if b.IsNil() {
		t.Error("expected false for non-nil alert")
	}
}

func TestAlert_EncodeFields(t *testing.T) {
	w := &recordingFieldWriter{fields: make(map[string]any)}

	var a *notify.Alert
	a.EncodeFields(w) // Should not panic and not write anything

	if len(w.fields) != 0 {
		t.Error("expected no fields written for nil alert")
	}

	b := &notify.Alert{
		Code:    "c1",
		Scope:   "s1",
		Subject: "sub1",
		Body:    "body1",
		Level:   msgtype.Warning,
	}

	b.EncodeFields(w)

	if w.fields["code"] != "c1" {
		t.Errorf("expected code c1, got %v", w.fields["code"])
	}
	if w.fields["scope"] != "s1" {
		t.Errorf("expected scope s1, got %v", w.fields["scope"])
	}
	if w.fields["subject"] != "sub1" {
		t.Errorf("expected subject sub1, got %v", w.fields["subject"])
	}
	if w.fields["body"] != "body1" {
		t.Errorf("expected body body1, got %v", w.fields["body"])
	}
	if w.fields["level"] != int64(msgtype.Warning) {
		t.Errorf("expected level %v, got %v", int64(msgtype.Warning), w.fields["level"])
	}
}

// testPayload implements model.Encodable for testing wrong payload types
type testPayload string

func (p testPayload) IsNil() bool { return false }
func (p testPayload) EncodeFields(w model.FieldWriter) {}

func TestEvents_Raise(t *testing.T) {
	broker := &mock.Broker{}

	// Invalid alert
	invalidAlert := notify.Alert{Code: ""}
	err := notify.Raise(broker, invalidAlert)
	if err != notify.ErrAlertCode {
		t.Errorf("expected ErrAlertCode, got %v", err)
	}

	// Valid alert
	validAlert := notify.Alert{Code: "test", Subject: "subj", Body: "body", Level: msgtype.Info}
	err = notify.Raise(broker, validAlert)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Note: We can't easily inspect published events without a subscriber,
	// so we verify this in TestEvents_Subscribe.
}

func TestEvents_Subscribe(t *testing.T) {
	broker := &mock.Broker{}

	var rcvd []notify.Alert
	h := func(a notify.Alert) {
		rcvd = append(rcvd, a)
	}

	var rcvdErr []error
	onError := func(err error) {
		rcvdErr = append(rcvdErr, err)
	}

	err := notify.Subscribe(broker, h, onError)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// subscribe nil handler
	err = notify.Subscribe(broker, nil, onError)
	if err != notify.ErrHandler {
		t.Errorf("expected ErrHandler, got %v", err)
	}

	// subscribe nil error handler
	err = notify.Subscribe(broker, h, nil)
	if err != notify.ErrHandler {
		t.Errorf("expected ErrHandler, got %v", err)
	}

	// Test second subscriber
	var rcvd2 []notify.Alert
	h2 := func(a notify.Alert) {
		rcvd2 = append(rcvd2, a)
	}
	notify.Subscribe(broker, h2, onError)

	validAlert := notify.Alert{Code: "test", Subject: "subj", Body: "body", Level: msgtype.Info}
	notify.Raise(broker, validAlert)

	// publish wrong payload
	broker.Publish(events.Event{Topic: notify.TopicAlert, Payload: testPayload("wrong payload")})

	if len(rcvd) != 1 || rcvd[0] != validAlert {
		t.Errorf("expected subscriber 1 to receive validAlert once, got %v", rcvd)
	}
	if len(rcvd2) != 1 || rcvd2[0] != validAlert {
		t.Errorf("expected subscriber 2 to receive validAlert once, got %v", rcvd2)
	}

	if len(rcvdErr) != 2 || rcvdErr[0] != notify.ErrPayload || rcvdErr[1] != notify.ErrPayload {
		t.Errorf("expected both subscribers to report ErrPayload, got %v", rcvdErr)
	}
}
