package notify

import "webtyp.com/lang"

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

// Field is one value a driver needs to build a channel. A screen renders one input per Field and
// stores it in Config.Settings (Secret == false) or Config.Secrets (Secret == true).
type Field struct {
	Key      string    // the Setting/Secret key the driver reads ("from", "api_token")
	Label    lang.Text // what a person reads above the input
	Help     lang.Text // optional: one line explaining where the value comes from
	Secret   bool      // true: the stored value is the NAME of a secret in the destination's store, never the value
	Required bool      // true: Registry.Load refuses a config that lacks it
}

// Driver builds channels of one kind. The returned Channel's Name() must equal c.Name.
type Driver interface {
	Kind() string
	Label() lang.Text // the kind as a person reads it ("Email")
	Fields() []Field  // every key New reads, in the order a screen shows them
	New(c Config, secrets SecretResolver) (Channel, error)
}
