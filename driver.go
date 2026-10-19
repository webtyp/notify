package notify

// Setting is one configuration value (a slice of these, never a map).
type Setting struct {
	Key   string
	Value string
}

// Config is the stored description of ONE channel instance: what the application keeps in its database.
type Config struct {
	Name     string // unique instance name ("pharmacy-email"); automations refer to this name
	Kind     string // the driver kind ("email", "whatsapp")
	Enabled  bool
	Settings []Setting // public settings (sender address, phone-number id). Never a secret value.
	Secrets  []Setting // Key = what the driver asks for; Value = the NAME of the secret in the destination's secret store, never its value
}

// SecretResolver reads a secret by name from the destination's secret store. The application injects it.
type SecretResolver interface {
	Secret(name string) (string, error)
}

// Driver builds channels of one kind. The returned Channel's Name() must equal c.Name.
type Driver interface {
	Kind() string
	New(c Config, secrets SecretResolver) (Channel, error)
}
