# notify
<img src="docs/img/badges.svg">

Alerts and delivery channels: the Channel port, recipients, and the rule that an automation without a configured channel is an error.

## The Contract

A business needs to send notifications (Alerts) to recipients when events occur. The `notify` library provides the core contract:
- `Alert`: what happened (with a `msgtype.Type` severity)
- `Source`: what produced the alert (the module and record ID)
- `Kind`: a catalog entry explaining an alert code
- `Area`: a business area an alert belongs to
- `Recipient`: who receives it
- `Channel`: how they receive it
- `Registry`: configured channels the application can use

## Catalog

Modules register areas and kinds at startup so a screen can list and explain every possible alert without free-text lookup.

```go
registry.RegisterArea(notify.Area{
	Code: "supplies",
	Name: lang.Text("Insumos"),
})

registry.RegisterKind(notify.Kind{
	Code: "stock.low",
	Name: lang.Text("Stock bajo"),
	Help: lang.Text("Se avisa cuando el total de un ítem cae bajo su mínimo"),
	Area: "supplies",
})
```

## Raising Alerts

Any module can raise an alert by using the `events.Publisher` it already has:

```go
notify.Raise(publisher, notify.Alert{
    Code: "stock.low",
    Level: msgtype.Warning,
    Subject: "Stock Low",
    Body: "Item 123 is below minimum.",
})
```

An alert manager then subscribes to these alerts to deliver them to the registry:

```go
subscription, err := notify.Subscribe(subscriber, func(a notify.Alert) {
    // Look up who needs to receive this alert, and which channels it should use
    // Then call registry.Deliver(...)
}, func(err error) {
    // Handle error
})
// subscription.Cancel() stops receiving alerts.
```

## Quick Start

```go
package main

import (
	"webtyp.com/msgtype"
	"webtyp.com/notify"
	"webtyp.com/notify/notifytest"
)

func main() {
	registry := &notify.Registry{}

	// Register channels
	registry.Register(&notifytest.Recorder{ChannelName: "email-main"})

	// Validate automations at start-up
	err := registry.Validate([]notify.Automation{
		{Code: "stock.low", Channels: []string{"email-main"}},
	})
	if err != nil {
		panic(err)
	}

	// Deliver an alert
	alert := notify.Alert{
		Code: "stock.low",
		Level: msgtype.Warning,
		Subject: "Stock Low",
		Body: "Item 123 is below minimum.",
		Source: notify.Ref{Type: "supply_item", ID: "123"},
	}
	recipient := notify.Recipient{ID: "usr1", Email: "admin@example.com"}

	results := registry.Deliver(alert, recipient, []string{"email-main"})
	for _, res := range results {
		if res.Err != nil {
			// handle delivery failure
		}
	}
}
```

## Dynamic Configurations and Drivers

Channels can also be loaded from dynamic configurations via drivers. A driver declares its label and the configuration fields it needs from the person setting it up:

```go
// Field is one value a driver needs to build a channel.
type Field struct {
	Key      string    // e.g. "from", "api_token"
	Label    lang.Text
	Help     lang.Text
	Secret   bool
	Required bool
}

type Driver interface {
	Kind() string
	Label() lang.Text
	Fields() []Field
	New(c Config, secrets SecretResolver) (Channel, error)
}
```

```go
// Loading channels from stored configurations without a restart
registry.Load([]notify.Config{
    {Name: "email-marketing", Kind: "email", Enabled: true},
}, secretResolver)
```

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for more details.
