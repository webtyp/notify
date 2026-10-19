//go:build !wasm

package notifytest

import (
	"sync"
	"webtyp.com/notify"
)

// Recorder is a Channel for tests: it records what it was asked to send and can be made to fail.
type Recorder struct {
	ChannelName string
	Fail        error // returned by every Send when not nil

	mu   sync.Mutex
	Sent []Sent
}

type Sent struct {
	Alert notify.Alert
	To    notify.Recipient
}

func (r *Recorder) Name() string {
	return r.ChannelName
}

func (r *Recorder) Send(a notify.Alert, to notify.Recipient) error {
	if r.Fail != nil {
		return r.Fail
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Sent = append(r.Sent, Sent{Alert: a, To: to})
	return nil
}
