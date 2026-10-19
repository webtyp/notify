# Architecture

## Why a missing channel is an error at start-up

In many notification systems, if a delivery channel is not configured, the message is silently dropped. This is dangerous because a business might rely on a notification (e.g., a critical stock alert or a billing failure) that nobody is actually receiving due to a missing or typo'd configuration.

To prevent this, the `notify` registry makes it a requirement that all channels needed by active automations must exist. When the application starts (or reloads configuration), it validates the required channels against the registered ones. If any are missing, it's a loud error.

## Concrete channels

A concrete channel (like `email` or `whatsapp`) lives in its own repository and implements the `Channel` interface. The `notify` library remains agnostic to the underlying transport mechanism or API.

## Dynamic Configurations and Drivers

Channels can be loaded dynamically from the database.
- A **Kind** refers to compiled code (a Driver) that knows how to create a specific type of channel (e.g., `email`, `whatsapp`).
- An **Instance** is a configured version of a kind, usually stored as a database row (represented by `Config`). For example, you might have multiple email channel instances for different purposes.

This separation allows you to add or modify instances (e.g., updating credentials or changing sender addresses) without rebuilding or restarting the application.

### Secret Resolution

Secrets are never stored in the database configurations. Instead, the `Config` refers to secrets by name, and the application provides a `SecretResolver` to fetch the actual values from a secure destination's secret store. This adheres to security best practices.

## No retry or queue here

The `notify` library only defines the contract. It does not implement retries or message queues. If a channel fails to send an alert, it returns an error. The responsibility of retrying, queueing, and deciding *when* to send belongs to the module managing the alerts (e.g., `alert_manager`).
