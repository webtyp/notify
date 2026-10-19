package notify

import (
	"webtyp.com/model"
	"webtyp.com/msgtype"
)

// Error is a failure of a notification rule. Constants, compared by value.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrChannelName      Error = "notify: a channel needs a name"
	ErrDuplicateChannel Error = "notify: a channel with that name is already registered"
	ErrChannelMissing   Error = "notify: the channel is not configured"
	ErrNoChannels       Error = "notify: an automation needs at least one channel"
	ErrNoAddress        Error = "notify: the recipient has no address for this channel"
	ErrAlertCode        Error = "notify: an alert needs a code"
	ErrAlertText        Error = "notify: an alert needs a subject and a body"
	ErrNoRecipient      Error = "notify: an alert needs a recipient"
	ErrDuplicateDriver  Error = "notify: a driver of that kind is already registered"
	ErrDriverMissing    Error = "notify: no driver for this kind of channel"
	ErrConfig           Error = "notify: invalid channel configuration"
	ErrAlertLevel       Error = "notify: an alert level must be Info, Warning or Error"
)

// Names of the channel kinds this ecosystem ships. The registry accepts any non-empty kind.
const (
	ChannelEmail    = "email"
	ChannelWhatsApp = "whatsapp"
	ChannelInApp    = "in_app"
)

// Recipient is who receives an alert. A channel uses the address it needs; ID is the recipient's id in its own module.
type Recipient struct {
	ID    string
	Name  string
	Email string
	Phone string // international format, digits with a leading +
}

// Alert is what happened that someone must know. Code identifies the rule that raised it ("stock.below_minimum");
// Scope is the area of the business it belongs to, so a report can show only what a person is responsible for.
type Alert struct {
	Code    string
	Scope   string
	Level   msgtype.Type
	Subject string
	Body    string
}

// Validate returns ErrAlertCode if Code is empty and ErrAlertText if Subject or Body is empty.
// It also returns ErrAlertLevel if Level is not Info, Warning, or Error.
func (a Alert) Validate() error {
	if a.Code == "" {
		return ErrAlertCode
	}
	if a.Subject == "" || a.Body == "" {
		return ErrAlertText
	}
	if a.Level != msgtype.Info && a.Level != msgtype.Warning && a.Level != msgtype.Error {
		return ErrAlertLevel
	}
	return nil
}

// IsNil implements model.Encodable.
func (a *Alert) IsNil() bool {
	return a == nil
}

// EncodeFields implements model.Encodable.
func (a *Alert) EncodeFields(w model.FieldWriter) {
	if a == nil {
		return
	}
	w.String("code", a.Code)
	w.String("scope", a.Scope)
	w.String("subject", a.Subject)
	w.String("body", a.Body)
	w.Int("level", int64(a.Level))
}

const TopicAlert = "notify.alert"

// Channel delivers an alert to one recipient. Send returns nil only if the delivery was accepted by the service behind
// the channel; any failure is returned, never swallowed. A Channel reads the address it needs from the Recipient and
// returns ErrNoAddress if it is empty.
type Channel interface {
	Name() string
	Send(a Alert, to Recipient) error
}

// Automation is a rule the application turned on (an alert code) together with the channel names it must be able to use.
type Automation struct {
	Code     string
	Channels []string
}

// Result is the outcome of delivering one alert through one channel.
type Result struct {
	Channel string
	Err     error // nil = accepted by the channel
}
