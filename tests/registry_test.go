//go:build !wasm

package notify_test

import (
	"strings"
	"sync"
	"testing"

	"webtyp.com/msgtype"
	"webtyp.com/notify"
	"webtyp.com/notify/notifytest"
	"webtyp.com/lang"
)

type dummySecretResolver struct{}

func (s dummySecretResolver) Secret(name string) (string, error) {
	return "secret_value", nil
}

type testDriver struct {
	kind     string
	failNew  bool
	nameFail bool
	fields   []notify.Field
}

func (d testDriver) Kind() string { return d.kind }
func (d testDriver) Label() lang.Text { return "Test" }
func (d testDriver) Fields() []notify.Field { return d.fields }
func (d testDriver) New(c notify.Config, secrets notify.SecretResolver) (notify.Channel, error) {
	if d.failNew {
		return nil, notify.ErrConfig
	}
	name := c.Name
	if d.nameFail {
		name = "wrong_name"
	}
	return &notifytest.Recorder{ChannelName: name}, nil
}

func TestRegistry_RegisterAndHas(t *testing.T) {
	r := &notify.Registry{}
	ch := &notifytest.Recorder{ChannelName: "test_ch"}

	if r.Has("test_ch") {
		t.Error("expected false for unregistered channel")
	}

	err := r.Register(ch)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !r.Has("test_ch") {
		t.Error("expected true for registered channel")
	}

	err = r.Register(ch)
	if err != notify.ErrDuplicateChannel {
		t.Errorf("expected ErrDuplicateChannel, got %v", err)
	}

	err = r.Register(&notifytest.Recorder{ChannelName: ""})
	if err != notify.ErrChannelName {
		t.Errorf("expected ErrChannelName, got %v", err)
	}
}

func TestRegistry_RegisterDriver(t *testing.T) {
	r := &notify.Registry{}
	err := r.RegisterDriver(testDriver{kind: "test_kind"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = r.RegisterDriver(testDriver{kind: "test_kind"})
	if err != notify.ErrDuplicateDriver {
		t.Errorf("expected ErrDuplicateDriver, got %v", err)
	}

	err = r.RegisterDriver(testDriver{kind: ""})
	if err != notify.ErrChannelName {
		t.Errorf("expected ErrChannelName, got %v", err)
	}
}

func TestRegistry_Load(t *testing.T) {
	r := &notify.Registry{}
	r.RegisterDriver(testDriver{kind: "working"})
	r.RegisterDriver(testDriver{kind: "failing", failNew: true})
	r.RegisterDriver(testDriver{kind: "wrong_name", nameFail: true})

	// Add a hardcoded channel to make sure it's preserved and checks for duplicates
	r.Register(&notifytest.Recorder{ChannelName: "hardcoded_ch"})

	configs := []notify.Config{
		{Name: "ch1", Kind: "working", Enabled: true},
		{Name: "ch2", Kind: "working", Enabled: false},         // Should be ignored
		{Name: "ch3", Kind: "missing_kind", Enabled: true},     // Error
		{Name: "ch4", Kind: "failing", Enabled: true},          // Error
		{Name: "ch5", Kind: "wrong_name", Enabled: true},       // Error
		{Name: "ch6", Kind: "working", Enabled: true},          // OK
		{Name: "ch6", Kind: "working", Enabled: true},          // Duplicate name in loaded
		{Name: "hardcoded_ch", Kind: "working", Enabled: true}, // Duplicate name with hardcoded
	}

	err := r.Load(configs, dummySecretResolver{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "ch3") || !strings.Contains(errMsg, "ch4") || !strings.Contains(errMsg, "ch5") || !strings.Contains(errMsg, "ch6") || !strings.Contains(errMsg, "hardcoded_ch") {
		t.Errorf("error message missing some failures: %v", errMsg)
	}

	if !r.Has("ch1") {
		t.Error("expected ch1 to be loaded")
	}
	if r.Has("ch2") {
		t.Error("expected ch2 not to be loaded")
	}
	if !r.Has("hardcoded_ch") {
		t.Error("expected hardcoded_ch to be preserved")
	}

	// Reload with a different config
	configs2 := []notify.Config{
		{Name: "ch_new", Kind: "working", Enabled: true},
	}
	err = r.Load(configs2, dummySecretResolver{})
	if err != nil {
		t.Fatalf("unexpected error on second load: %v", err)
	}

	if r.Has("ch1") {
		t.Error("expected ch1 to be removed")
	}
	if !r.Has("ch_new") {
		t.Error("expected ch_new to be loaded")
	}
	if !r.Has("hardcoded_ch") {
		t.Error("expected hardcoded_ch to be preserved")
	}
}

func TestRegistry_Validate(t *testing.T) {
	r := &notify.Registry{}
	r.Register(&notifytest.Recorder{ChannelName: "ch1"})

	err := r.Validate([]notify.Automation{})
	if err != nil {
		t.Errorf("unexpected error for empty automations: %v", err)
	}

	err = r.Validate([]notify.Automation{
		{Code: "auto1", Channels: []string{"ch1"}},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = r.Validate([]notify.Automation{
		{Code: "auto1", Channels: []string{"ch1", "missing"}},
		{Code: "auto2", Channels: []string{}},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "missing") || !strings.Contains(errMsg, string(notify.ErrChannelMissing)) {
		t.Errorf("expected error about missing channel, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "auto2") || !strings.Contains(errMsg, string(notify.ErrNoChannels)) {
		t.Errorf("expected error about no channels, got: %v", errMsg)
	}
}

func TestRegistry_Deliver(t *testing.T) {
	r := &notify.Registry{}
	ch1 := &notifytest.Recorder{ChannelName: "ch1"}
	ch2 := &notifytest.Recorder{ChannelName: "ch2", Fail: notify.ErrNoAddress}
	r.Register(ch1)
	r.Register(ch2)

	alert := notify.Alert{Code: "test", Subject: "subj", Body: "body", Level: msgtype.Info}
	recip := notify.Recipient{ID: "r1"}

	// No channels
	res := r.Deliver(alert, recip, []string{})
	if len(res) != 1 || res[0].Err != notify.ErrNoChannels {
		t.Errorf("expected single ErrNoChannels, got: %v", res)
	}

	// Invalid alert
	invalidAlert := notify.Alert{Code: ""}
	res = r.Deliver(invalidAlert, recip, []string{"ch1", "ch2"})
	if len(res) != 2 || res[0].Err != notify.ErrAlertCode || res[1].Err != notify.ErrAlertCode {
		t.Errorf("expected all ErrAlertCode, got: %v", res)
	}
	if len(ch1.Sent) > 0 {
		t.Error("expected no delivery for invalid alert")
	}

	// Normal delivery
	res = r.Deliver(alert, recip, []string{"ch1", "missing", "ch2"})
	if len(res) != 3 {
		t.Fatalf("expected 3 results, got %d", len(res))
	}

	if res[0].Channel != "ch1" || res[0].Err != nil {
		t.Errorf("unexpected result 0: %v", res[0])
	}
	if res[1].Channel != "missing" || res[1].Err != notify.ErrChannelMissing {
		t.Errorf("unexpected result 1: %v", res[1])
	}
	if res[2].Channel != "ch2" || res[2].Err != notify.ErrNoAddress {
		t.Errorf("unexpected result 2: %v", res[2])
	}

	if len(ch1.Sent) != 1 || ch1.Sent[0].Alert.Code != "test" {
		t.Errorf("expected 1 sent to ch1, got %v", ch1.Sent)
	}
	if len(ch2.Sent) != 0 {
		t.Errorf("expected 0 sent to ch2, got %v", ch2.Sent)
	}
}

func TestRegistry_ConcurrentDeliverLoad(t *testing.T) {
	r := &notify.Registry{}
	r.RegisterDriver(testDriver{kind: "working"})
	r.Load([]notify.Config{{Name: "ch", Kind: "working", Enabled: true}}, dummySecretResolver{})

	alert := notify.Alert{Code: "test", Subject: "subj", Body: "body", Level: msgtype.Info}
	recip := notify.Recipient{ID: "r1"}

	var wg sync.WaitGroup

	// Start concurrent deliveries
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Deliver(alert, recip, []string{"ch"})
		}()
	}

	// Start concurrent loads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Load([]notify.Config{{Name: "ch", Kind: "working", Enabled: true}}, dummySecretResolver{})
		}()
	}

	wg.Wait()
}

func TestRegistry_DriverFields(t *testing.T) {
	r := &notify.Registry{}

	// OK
	err := r.RegisterDriver(testDriver{kind: "email", fields: []notify.Field{{Key: "from", Label: "Sender"}}})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Key: "" -> ErrFieldKey
	err = r.RegisterDriver(testDriver{kind: "sms", fields: []notify.Field{{Key: "", Label: "Sender"}}})
	if err != notify.ErrFieldKey {
		t.Errorf("expected ErrFieldKey, got %v", err)
	}

	// Duplicate Key -> ErrDuplicateField
	err = r.RegisterDriver(testDriver{kind: "sms", fields: []notify.Field{{Key: "from", Label: "A"}, {Key: "from", Label: "B"}}})
	if err != notify.ErrDuplicateField {
		t.Errorf("expected ErrDuplicateField, got %v", err)
	}

	// Label: "" -> ErrFieldLabel
	err = r.RegisterDriver(testDriver{kind: "sms", fields: []notify.Field{{Key: "from", Label: ""}}})
	if err != notify.ErrFieldLabel {
		t.Errorf("expected ErrFieldLabel, got %v", err)
	}

	r.RegisterDriver(testDriver{kind: "whatsapp"})

	kinds := r.DriverKinds()
	if len(kinds) != 2 || kinds[0] != "email" || kinds[1] != "whatsapp" {
		t.Errorf("expected [email whatsapp], got %v", kinds)
	}

	lbl, fields, ok := r.DriverOf("email")
	if !ok || lbl != "Test" || len(fields) != 1 || fields[0].Key != "from" {
		t.Errorf("DriverOf(email) unexpected result: %v %v %v", lbl, fields, ok)
	}

	_, _, ok = r.DriverOf("sms")
	if ok {
		t.Errorf("expected ok=false for sms")
	}

	// Load with missing required field
	r2 := &notify.Registry{}
	r2.RegisterDriver(testDriver{
		kind: "req",
		fields: []notify.Field{{Key: "needed", Label: "Needed", Required: true}},
		failNew: true, // we want to make sure it fails before calling New
	})

	err = r2.Load([]notify.Config{
		{Name: "c1", Kind: "req", Enabled: true},
	}, dummySecretResolver{})

	if err == nil || !strings.Contains(err.Error(), "c1: notify: invalid channel configuration: missing needed") {
		t.Errorf("expected missing field error, got %v", err)
	}

	// Load with required present
	r3 := &notify.Registry{}
	r3.RegisterDriver(testDriver{
		kind: "req",
		fields: []notify.Field{{Key: "needed", Label: "Needed", Required: true}},
	})

	err = r3.Load([]notify.Config{
		{Name: "c2", Kind: "req", Enabled: true, Settings: []notify.Setting{{Key: "needed", Value: "present"}}},
	}, dummySecretResolver{})

	if err != nil {
		t.Errorf("expected no error when required present, got %v", err)
	}
}
