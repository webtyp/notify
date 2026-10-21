package notify

import (
	"sync"
	"webtyp.com/fmt"
	"webtyp.com/lang"
)

// Registry holds the channels the application configured and the drivers to build them.
type Registry struct {
	mu        sync.RWMutex
	hardcoded []Channel
	loaded    []Channel
	drivers   []Driver
	areas     []Area
	kinds     []Kind
}

// Register adds a channel directly. An empty name → ErrChannelName; a name already registered → ErrDuplicateChannel.
func (r *Registry) Register(c Channel) error {
	if c.Name() == "" {
		return ErrChannelName
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.hardcoded {
		if existing.Name() == c.Name() {
			return ErrDuplicateChannel
		}
	}
	// A name already built from a stored configuration is taken as well.
	for _, existing := range r.loaded {
		if existing.Name() == c.Name() {
			return ErrDuplicateChannel
		}
	}

	r.hardcoded = append(r.hardcoded, c)
	return nil
}

// RegisterArea adds an area. An empty code → ErrAreaCode; a code already registered → ErrDuplicateArea.
func (r *Registry) RegisterArea(a Area) error {
	if a.Code == "" {
		return ErrAreaCode
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.areas {
		if existing.Code == a.Code {
			return ErrDuplicateArea
		}
	}

	r.areas = append(r.areas, a)
	return nil
}

// RegisterKind adds a kind. An empty code → ErrKindCode; an empty name → ErrKindText; a code already registered → ErrDuplicateKind.
// A kind with no area is only allowed if no areas are registered at all.
func (r *Registry) RegisterKind(k Kind) error {
	if k.Code == "" {
		return ErrKindCode
	}
	if string(k.Name) == "" { // lang.Text is a string alias, but check just in case
		return ErrKindText
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.kinds {
		if existing.Code == k.Code {
			return ErrDuplicateKind
		}
	}

	if len(r.areas) > 0 {
		foundArea := false
		for _, area := range r.areas {
			if area.Code == k.Area {
				foundArea = true
				break
			}
		}
		if !foundArea {
			return ErrAreaMissing
		}
	} else if k.Area != "" {
		return ErrAreaMissing
	}

	r.kinds = append(r.kinds, k)
	return nil
}

// Kinds returns a snapshot of all registered kinds.
func (r *Registry) Kinds() []Kind {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kinds := make([]Kind, len(r.kinds))
	copy(kinds, r.kinds)
	return kinds
}

// Areas returns a snapshot of all registered areas.
func (r *Registry) Areas() []Area {
	r.mu.RLock()
	defer r.mu.RUnlock()
	areas := make([]Area, len(r.areas))
	copy(areas, r.areas)
	return areas
}

// KindOf looks up a kind by its code.
func (r *Registry) KindOf(code string) (Kind, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, k := range r.kinds {
		if k.Code == code {
			return k, true
		}
	}
	return Kind{}, false
}

// RegisterDriver adds a driver. An empty kind → ErrChannelName; a kind already registered → ErrDuplicateDriver.
func (r *Registry) RegisterDriver(d Driver) error {
	if d.Kind() == "" {
		return ErrChannelName
	}
	fields := d.Fields()
	for i, f := range fields {
		if f.Key == "" {
			return ErrFieldKey
		}
		if f.Label == "" {
			return ErrFieldLabel
		}
		for j := 0; j < i; j++ {
			if fields[j].Key == f.Key {
				return ErrDuplicateField
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.drivers {
		if existing.Kind() == d.Kind() {
			return ErrDuplicateDriver
		}
	}

	r.drivers = append(r.drivers, d)
	return nil
}

// DriverKinds returns the registered driver kinds, in registration order.
func (r *Registry) DriverKinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kinds := make([]string, len(r.drivers))
	for i, d := range r.drivers {
		kinds[i] = d.Kind()
	}
	return kinds
}

// DriverOf returns the label and fields of a registered driver kind; ok is false if none.
func (r *Registry) DriverOf(kind string) (label lang.Text, fields []Field, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.drivers {
		if d.Kind() == kind {
			return d.Label(), d.Fields(), true
		}
	}
	return "", nil, false
}

// Has reports whether a channel with that name is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.hardcoded {
		if c.Name() == name {
			return true
		}
	}
	for _, c := range r.loaded {
		if c.Name() == name {
			return true
		}
	}
	return false
}

func (r *Registry) getChannel(name string) Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.hardcoded {
		if c.Name() == name {
			return c
		}
	}
	for _, c := range r.loaded {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// Load rebuilds the whole channel set from the stored configs. Only enabled configs are built.
// The new set replaces the old one atomically.
func (r *Registry) Load(configs []Config, secrets SecretResolver) error {
	r.mu.RLock()
	drivers := append([]Driver(nil), r.drivers...)
	hardcoded := append([]Channel(nil), r.hardcoded...)
	r.mu.RUnlock()

	var newLoaded []Channel
	var errors []string

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		var driver Driver
		for _, d := range drivers {
			if d.Kind() == config.Kind {
				driver = d
				break
			}
		}

		if driver == nil {
			errors = append(errors, fmt.Sprintf("%s: %s", config.Name, ErrDriverMissing))
			continue
		}

		var missingKey string
		for _, f := range driver.Fields() {
			if !f.Required {
				continue
			}
			found := false
			settings := config.Settings
			if f.Secret {
				settings = config.Secrets
			}
			for _, s := range settings {
				if s.Key == f.Key && s.Value != "" {
					found = true
					break
				}
			}
			if !found {
				missingKey = f.Key
				break
			}
		}

		if missingKey != "" {
			errors = append(errors, fmt.Sprintf("%s: %s: missing %s", config.Name, ErrConfig, missingKey))
			continue
		}

		ch, err := driver.New(config, secrets)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s: %s", config.Name, ErrConfig, err))
			continue
		}

		if ch.Name() == "" || ch.Name() != config.Name {
			errors = append(errors, fmt.Sprintf("%s: %s", config.Name, ErrConfig))
			continue
		}

		// Check for duplicate names among newly loaded and hardcoded
		duplicate := false
		for _, existing := range newLoaded {
			if existing.Name() == ch.Name() {
				duplicate = true
				break
			}
		}
		if !duplicate {
			for _, existing := range hardcoded {
				if existing.Name() == ch.Name() {
					duplicate = true
					break
				}
			}
		}

		if duplicate {
			errors = append(errors, fmt.Sprintf("%s: %s", config.Name, ErrConfig))
			continue
		}

		newLoaded = append(newLoaded, ch)
	}

	r.mu.Lock()
	r.loaded = newLoaded
	r.mu.Unlock()

	if len(errors) > 0 {
		msg := "load failed:"
		for _, e := range errors {
			msg += "\n" + e
		}
		return Error(msg)
	}

	return nil
}

// Validate checks every automation against the registry.
func (r *Registry) Validate(autos []Automation) error {
	var errors []string
	for _, auto := range autos {
		if len(auto.Channels) == 0 {
			errors = append(errors, fmt.Sprintf("%s: %s", auto.Code, ErrNoChannels))
			continue
		}
		for _, chName := range auto.Channels {
			if !r.Has(chName) {
				errors = append(errors, fmt.Sprintf("%s: %s: %s", auto.Code, chName, ErrChannelMissing))
			}
		}
	}

	if len(errors) > 0 {
		msg := "validation failed:"
		for _, e := range errors {
			msg += "\n" + e
		}
		return Error(msg)
	}
	return nil
}

// Deliver sends the alert to the recipient through each named channel.
func (r *Registry) Deliver(a Alert, to Recipient, channels []string) []Result {
	if len(channels) == 0 {
		return []Result{{Err: ErrNoChannels}}
	}

	errAlert := a.Validate()
	if errAlert != nil {
		results := make([]Result, len(channels))
		for i, chName := range channels {
			results[i] = Result{Channel: chName, Err: errAlert}
		}
		return results
	}

	results := make([]Result, len(channels))
	for i, chName := range channels {
		ch := r.getChannel(chName)
		if ch == nil {
			results[i] = Result{Channel: chName, Err: ErrChannelMissing}
			continue
		}

		errSend := ch.Send(a, to)
		results[i] = Result{Channel: chName, Err: errSend}
	}

	return results
}
