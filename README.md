# Logger

`logger` is a thread-safe, high-performance structured console and file logging package for Go. It supports clean ANSI colorized outputs, log level filtering, custom independent loggers, customizable log date layouts, asynchronous Grafana Loki shipping, and Prometheus metrics tracking.

## Installation

```bash
go get github.com/danmaina/logger/v2
```

## Import

```go
import "github.com/danmaina/logger/v2"
```

## Configuration Guide

The package can be configured either as a single global logger, or by instantiating independent custom loggers.

### 1. Configuring the Global Logger

By default, the package-level log functions (e.g., `logger.Info`, `logger.Warn`, `logger.Err`, `logger.Fatal`) write to a thread-safe, global `defaultLogger` instance. This global instance starts with the log level set to `LevelDebug` (2), sending colorized logs to standard error.

To configure the global logger:

```go
package main

import (
    "time"
    "github.com/danmaina/logger/v2"
    "github.com/prometheus/client_golang/prometheus"
)

func main() {
    // A. Configure File Logging Target
    // If set, the logger automatically creates this directory and writes level-specific log files.
    logger.SetLogsDirectory("/var/log/myapp")

    // B. Configure Log Level Filter
    // Only logs at or below the configured level will be printed/stored.
    // Available: LevelError=0, LevelInfo=1, LevelDebug=2
    logger.SetLogLevel(int(logger.LevelInfo))

    // C. Configure Log Date/Time Layout (Optional)
    // Allows setting custom Go time formats (e.g., time.RFC3339, "15:04:05").
    // Set to "" to completely disable timestamps in both console and file log outputs.
    logger.SetTimeLayout("2006-01-02 15:04:05.000")

    // D. Configure Grafana Loki Shipping (Optional)
    logger.SetLokiConfig(logger.LokiConfig{
        URL:           "http://localhost:3100",
        Labels:        map[string]string{"app": "my-service", "env": "production"},
        BatchWait:     1 * time.Second,
        BatchCapacity: 100,
    })
    defer logger.Close() // Ensure Loki batches are flushed on shutdown

    // E. Configure Prometheus Metrics (Optional)
    _ = logger.EnablePrometheus(prometheus.DefaultRegisterer)

    // Write logs (the entire log line on console is color-coded)
    logger.Info("Global logger configured successfully!")
}
```

### 2. Creating & Configuring Custom Loggers

For applications requiring different log destinations or levels for different components (e.g. database logs vs. HTTP request logs), you can instantiate custom `Logger` structs. Each instance is completely independent and thread-safe.

```go
package main

import (
    "github.com/danmaina/logger/v2"
)

func main() {
    // Create an audit logger that only captures errors/fatal logs and stores them in "/var/log/audit"
    auditLogger := logger.New(logger.LevelError, "/var/log/audit")
    auditLogger.SetTimeLayout("") // Disable timestamps for audit logs

    // Create an application runner logger that logs errors, warn, and info to "/var/log/app"
    appLogger := logger.New(logger.LevelInfo, "/var/log/app")

    // Log to their respective targets
    auditLogger.Err("Database connection failed")
    appLogger.Info("Application startup sequence initiated")
}
```

---

## Log Levels & Colorization

The package exposes exactly three configurable logging levels. Individual log functions route to their target log files and utilize console prefixes/colorization as shown below. On the console, the **entire log line** (including the level prefix and date timestamp) is color-coded to maximize visual clarity:

| Severity Level | Configured Logging Level | Target File | Prefix | Console Color |
|---|---|---|---|---|
| **Fatal** | `logger.LevelError` (0) | `FATAL.log` | `[FATAL] \|\| ` | Red (Full Line) |
| **Error** | `logger.LevelError` (0) | `ERROR.log` | `[ERROR] \|\| ` | Red (Full Line) |
| **Warn** | `logger.LevelInfo` (1) | `WARN.log` | `[WARN] \|\| ` | Yellow (Full Line) |
| **Info** | `logger.LevelInfo` (1) | `INFO.log` | `[INFO] \|\| ` | Blue (Full Line) |
| **Debug** | `logger.LevelDebug` (2) | `DEBUG.log` | `[DEBUG] \|\| ` | Cyan (Full Line) |

---

## Features

### Grafana Loki Shipping
Streams log messages asynchronously in JSON batches to Grafana Loki `/loki/api/v1/push`.
- **Batching & Buffering**: Logs are buffered internally and shipped in batches based on `BatchWait` (period) and `BatchCapacity` (capacity) settings.
- **Resilience**: Pushes run in a background worker goroutine to prevent logging from blocking application execution.

### Prometheus Metrics
Exposes a `logger_messages_total` counter vector partitioned by the `level` label. Excellent for building Grafana alert dashboards on error spikes.

---

## License

MIT