package notify

import (
	"sync"
	"webtyp.com/fmt"
)

// Registry holds the channels the application configured and the drivers to build them.
type Registry struct {
	mu        sync.RWMutex
	hardcoded []Channel
	loaded    []Channel
	drivers   []Driver
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
	// Note: We check if it conflicts with loaded ones as well
	for _, existing := range r.loaded {
		if existing.Name() == c.Name() {
			return ErrDuplicateChannel
		}
	}

	r.hardcoded = append(r.hardcoded, c)
	return nil
}

// RegisterDriver adds a driver. An empty kind → ErrChannelName; a kind already registered → ErrDuplicateDriver.
func (r *Registry) RegisterDriver(d Driver) error {
	if d.Kind() == "" {
		return ErrChannelName // Assuming ErrChannelName is used for empty kind based on prompt "empty kind → ErrChannelName"
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

		ch, err := driver.New(config, secrets)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", config.Name, ErrConfig))
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
		return Error(msg) // Need a way to return multiple errors as one Error. Returning custom Error.
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
