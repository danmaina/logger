# Release Notes - v1.1.0

This release introduces significant enhancements to the `logger` package, including concurrency safety, simplified log levels, custom date formatting, full console line color coding, support for Grafana Loki shipping, Prometheus metrics integration, and full package documentation.

## Features

- **Full Log Line Color Coding**: Stderr console output now color-codes the entire log line (prefix, date timestamp, and log message) in a single color block for improved visual tracking.
- **Configurable Log Date/Time Layout**: Control the format of log timestamps via `logger.SetTimeLayout(layout)` using standard Go layout formats (e.g. `time.RFC3339`). Omitting the layout (`""`) disables timestamps in all logs.
- **Simplified Log Levels**: Configurable logging levels are now restricted to exactly three levels:
  - `LevelError` (0): Logs `ERROR` and `FATAL` severity messages.
  - `LevelInfo` (1): Logs `ERROR`, `FATAL`, `WARN`, and `INFO` severity messages.
  - `LevelDebug` (2): Logs all messages (`ERROR`, `FATAL`, `WARN`, `INFO`, and `DEBUG`).
- **Loki Log Shipping**: Asynchronously stream log messages in batches to Grafana Loki `/loki/api/v1/push` endpoint.
- **Prometheus Metrics**: Register and increment `logger_messages_total` counter partitioned by log level (`level`).
- **WARN level support**: Expose new `WARN`, `Warn`, and `Warnf` logging functions.
- **Improved API surface**: Added camelCase (`Info`, `Warn`, etc.) and formatted logging (`Infof`, `Warnf`, etc.) functions.
- **Fully Thread-Safe**: Restructured state storage using `sync.RWMutex` to secure package-level variables and local structures.

## Fixes

- **Formatting Correction**: Restructured formatting logic to prevent printing arguments as raw slice brackets (e.g. `[message]`) on standard outputs and files.
- **Global Log State Isolation**: Replaced modification of standard library `log` package state with separate `*log.Logger` configurations.

## Setup & Configuration

Refer to [README.md](README.md) for usage and configuration examples.
