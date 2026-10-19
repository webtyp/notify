# notify
<img src="docs/img/badges.svg">

Alerts and delivery channels: the Channel port, recipients, and the rule that an automation without a configured channel is an error.

## The Contract

A business needs to send notifications (Alerts) to recipients when events occur. The `notify` library provides the core contract:
- `Alert`: what happened
- `Recipient`: who receives it
- `Channel`: how they receive it
- `Registry`: configured channels the application can use

## Quick Start

```go
package main

import (
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
		Subject: "Stock Low",
		Body: "Item 123 is below minimum.",
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

## Dynamic Configurations

Channels can also be loaded from dynamic configurations via drivers.

```go
// Loading channels from stored configurations without a restart
registry.Load([]notify.Config{
    {Name: "email-marketing", Kind: "email", Enabled: true},
}, secretResolver)
```

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for more details.
