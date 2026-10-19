package notify

import "webtyp.com/events"

const (
	ErrHandler Error = "notify: Subscribe needs a handler and an error handler"
	ErrPayload Error = "notify: the payload of a notify.alert event is not an alert"
)

// Raise announces an alert on TopicAlert through the module's events.Publisher.
// An invalid alert (Alert.Validate) is NOT published: its error is returned.
func Raise(p events.Publisher, a Alert) error {
	if err := a.Validate(); err != nil {
		return err
	}
	p.Publish(events.Event{Topic: TopicAlert, Payload: &a})
	return nil
}

// Subscribe registers h to receive every alert raised on TopicAlert. A nil h or nil onError → the new constant
// ErrHandler = "notify: Subscribe needs a handler and an error handler" (an error is never dropped silently).
// A payload on TopicAlert that is not an *Alert goes to onError(ErrPayload) with
// ErrPayload = "notify: the payload of a notify.alert event is not an alert".
func Subscribe(s events.Subscriber, h func(Alert), onError func(error)) error {
	if h == nil || onError == nil {
		return ErrHandler
	}
	s.Subscribe(TopicAlert, func(e events.Event) {
		a, ok := e.Payload.(*Alert)
		if !ok || a == nil {
			onError(ErrPayload)
			return
		}
		h(*a)
	})
	return nil
}
