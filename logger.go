/*
Package logger provides a thread-safe, structured console and file logging utility for Go applications.
It features multiple log levels, clean ANSI colorized console outputs, structured file output per log level,
asynchronous batch shipping to Grafana Loki, and Prometheus metric tracking.

Log Levels:
  - LevelError (0): Logs only ERROR and FATAL messages.
  - LevelInfo (1): Logs ERROR, FATAL, WARN, and INFO messages.
  - LevelDebug (2): Logs ERROR, FATAL, WARN, INFO, and DEBUG messages.

Usage:
Initialize the logger directory to begin writing logs to specific files:

	logger.SetLogsDirectory("/var/log/app")

Or configure Loki integration to ship logs:

	logger.SetLokiConfig(logger.LokiConfig{
		URL: "http://localhost:3100",
		Labels: map[string]string{"app": "my-service"},
	})

And configure Prometheus metrics integration:

	logger.EnablePrometheus(prometheus.DefaultRegisterer)
*/
package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Level int

const (
	LevelError Level = iota // 0
	LevelInfo               // 1
	LevelDebug              // 2
)

const (
	infoLogFile  = "INFO.log"
	warnLogFile  = "WARN.log"
	debugLogFile = "DEBUG.log"
	errorLogFile = "ERROR.log"
	fatalLogFile = "FATAL.log"

	infoLogPrefix  = "[INFO] || "
	warnLogPrefix  = "[WARN] || "
	errorLogPrefix = "[ERROR] || "
	debugLogPrefix = "[DEBUG] || "
	fatalLogPrefix = "[FATAL] || "

	infoColor  = "\033[1;34m%v\033[0m"
	warnColor  = "\033[1;33m%v\033[0m"
	errorColor = "\033[1;31m%v\033[0m"
	debugColor = "\033[0;36m%v\033[0m"
)

// LokiConfig holds configurations required for shipping logs to Grafana Loki.
type LokiConfig struct {
	URL           string            // URL of Loki server, e.g. "http://localhost:3100"
	Labels        map[string]string // Labels applied to every stream
	BatchWait     time.Duration     // Max wait time before sending batch. Defaults to 1s if <= 0.
	BatchCapacity int               // Max batch size before sending. Defaults to 100 if <= 0.
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiEntry struct {
	timestamp time.Time
	line      string
}

type lokiClient struct {
	url           string
	labels        map[string]string
	batchWait     time.Duration
	batchCapacity int
	ch            chan lokiEntry
	closeCh       chan struct{}
	wg            sync.WaitGroup
	httpClient    *http.Client
}

func newLokiClient(cfg LokiConfig) *lokiClient {
	batchWait := cfg.BatchWait
	if batchWait <= 0 {
		batchWait = 1 * time.Second
	}
	batchCapacity := cfg.BatchCapacity
	if batchCapacity <= 0 {
		batchCapacity = 100
	}

	c := &lokiClient{
		url:           cfg.URL,
		batchWait:     batchWait,
		batchCapacity: batchCapacity,
		ch:            make(chan lokiEntry, 1000),
		closeCh:       make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	c.labels = make(map[string]string)
	for k, v := range cfg.Labels {
		c.labels[k] = v
	}

	c.wg.Add(1)
	go c.run()

	return c
}

func (c *lokiClient) send(entry lokiEntry) {
	select {
	case c.ch <- entry:
	default:
		log.Println("Loki push channel full, dropping log line:", entry.line)
	}
}

func (c *lokiClient) run() {
	defer c.wg.Done()

	var entries []lokiEntry
	ticker := time.NewTicker(c.batchWait)
	defer ticker.Stop()

	flush := func() {
		if len(entries) == 0 {
			return
		}
		c.push(entries)
		entries = nil
	}

	for {
		select {
		case entry, ok := <-c.ch:
			if !ok {
				flush()
				return
			}
			entries = append(entries, entry)
			if len(entries) >= c.batchCapacity {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-c.closeCh:
			for {
				select {
				case entry, ok := <-c.ch:
					if !ok {
						flush()
						return
					}
					entries = append(entries, entry)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (c *lokiClient) push(entries []lokiEntry) {
	values := make([][]string, len(entries))
	for i, entry := range entries {
		ns := entry.timestamp.UnixNano()
		values[i] = []string{
			fmt.Sprintf("%d", ns),
			entry.line,
		}
	}

	reqBody := lokiPushRequest{
		Streams: []lokiStream{
			{
				Stream: c.labels,
				Values: values,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("Failed to marshal Loki push request:", err)
		return
	}

	pushURL := c.url
	if !strings.HasSuffix(pushURL, "/loki/api/v1/push") {
		pushURL = strings.TrimSuffix(pushURL, "/") + "/loki/api/v1/push"
	}

	req, err := http.NewRequest("POST", pushURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Println("Failed to create Loki HTTP request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println("Failed to send logs to Loki:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Loki push request failed with status: %s", resp.Status)
	}
}

func (c *lokiClient) close() {
	close(c.closeCh)
	c.wg.Wait()
}

// Logger represents a thread-safe logger configuration.
type Logger struct {
	mu                sync.RWMutex
	level             Level
	outputDir         string
	timeLayout        string
	lokiClient        *lokiClient
	prometheusEnabled bool
}

// defaultLogger is the global default logger instance.
var defaultLogger = &Logger{
	level:      LevelDebug, // Default to debug (logs all logs by default)
	timeLayout: "2006/01/02 15:04:05",
}

var (
	logCounterVec *prometheus.CounterVec
	promOnce      sync.Once
)

func initPrometheus() {
	promOnce.Do(func() {
		logCounterVec = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "logger_messages_total",
				Help: "Total number of log messages processed by logger, partitioned by level.",
			},
			[]string{"level"},
		)
	})
}

// SetLogsDirectory sets the directory where log files will be created for the default logger.
func SetLogsDirectory(dir string) {
	defaultLogger.SetLogsDirectory(dir)
}

// SetLogLevel sets the log level for the default logger.
func SetLogLevel(level int) {
	defaultLogger.SetLogLevel(level)
}

// SetTimeLayout sets the log date format layout for the default logger.
func SetTimeLayout(layout string) {
	defaultLogger.SetTimeLayout(layout)
}

// SetLokiConfig configures and starts Loki shipper for the default logger.
func SetLokiConfig(cfg LokiConfig) {
	defaultLogger.SetLokiConfig(cfg)
}

// EnablePrometheus enables prometheus metrics reporting for the default logger.
func EnablePrometheus(registerer prometheus.Registerer) error {
	return defaultLogger.EnablePrometheus(registerer)
}

// Close flushes and stops the default logger background processes.
func Close() error {
	return defaultLogger.Close()
}

// SetLogsDirectory sets the directory where log files will be created for this Logger.
func (l *Logger) SetLogsDirectory(dir string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Println("Could not set the log directory: ", err)
		return
	}
	l.outputDir = dir
}

// SetLogLevel sets the log level for this Logger.
func (l *Logger) SetLogLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < 0 {
		level = 0
	} else if level > 2 {
		level = 2
	}
	l.level = Level(level)
}

// SetTimeLayout sets the log date format layout for this Logger.
func (l *Logger) SetTimeLayout(layout string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeLayout = layout
}

// SetLokiConfig configures and starts Loki log shipping worker for this Logger.
func (l *Logger) SetLokiConfig(cfg LokiConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lokiClient != nil {
		l.lokiClient.close()
		l.lokiClient = nil
	}

	if cfg.URL != "" {
		l.lokiClient = newLokiClient(cfg)
	}
}

// EnablePrometheus registers and enables Prometheus metrics for this Logger.
func (l *Logger) EnablePrometheus(registerer prometheus.Registerer) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	initPrometheus()

	err := registerer.Register(logCounterVec)
	if err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			return err
		}
	}
	l.prometheusEnabled = true
	return nil
}

// Close flushes and shuts down background log shipping worker.
func (l *Logger) Close() error {
	l.mu.Lock()
	client := l.lokiClient
	l.lokiClient = nil
	l.mu.Unlock()

	if client != nil {
		client.close()
	}
	return nil
}

// New creates a new Logger instance with the specified log level and output directory.
func New(level Level, outputDir string) *Logger {
	return &Logger{
		level:      level,
		outputDir:  outputDir,
		timeLayout: "2006/01/02 15:04:05",
	}
}

// logToFile writes a formatted message to a specific file in the output directory.
func (l *Logger) logToFile(filename string, prefix string, timestamp string, msg string) error {
	l.mu.RLock()
	dir := l.outputDir
	l.mu.RUnlock()

	if dir == "" {
		return nil
	}

	filePath := filepath.Join(dir, filename)

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	line := prefix + timestamp + msg + "\n"
	_, err = f.WriteString(line)
	return err
}

// logMsg formats and logs a message if the level is enabled.
func (l *Logger) logMsg(lvl Level, label string, filename string, prefix string, colorFormat string, v ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	dir := l.outputDir
	layout := l.timeLayout
	loki := l.lokiClient
	prom := l.prometheusEnabled
	l.mu.RUnlock()

	if lvl > currentLevel {
		return
	}

	msg := fmt.Sprint(v...)

	var timestamp string
	if layout != "" {
		timestamp = time.Now().Format(layout) + " "
	}

	if dir != "" {
		if err := l.logToFile(filename, prefix, timestamp, msg); err != nil {
			log.Printf("Could not log to file %s: %v", filename, err)
		}
	}

	fullLine := prefix + timestamp + msg

	if loki != nil {
		loki.send(lokiEntry{
			timestamp: time.Now(),
			line:      fullLine,
		})
	}

	if prom {
		logCounterVec.WithLabelValues(label).Inc()
	}

	// Print the complete log line color-coded to stderr
	fmt.Fprintf(os.Stderr, colorFormat+"\n", fullLine)
}

// logMsgf formats and logs a message using formatting directives if the level is enabled.
func (l *Logger) logMsgf(lvl Level, label string, filename string, prefix string, colorFormat string, format string, v ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	dir := l.outputDir
	layout := l.timeLayout
	loki := l.lokiClient
	prom := l.prometheusEnabled
	l.mu.RUnlock()

	if lvl > currentLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)

	var timestamp string
	if layout != "" {
		timestamp = time.Now().Format(layout) + " "
	}

	if dir != "" {
		if err := l.logToFile(filename, prefix, timestamp, msg); err != nil {
			log.Printf("Could not log to file %s: %v", filename, err)
		}
	}

	fullLine := prefix + timestamp + msg

	if loki != nil {
		loki.send(lokiEntry{
			timestamp: time.Now(),
			line:      fullLine,
		})
	}

	if prom {
		logCounterVec.WithLabelValues(label).Inc()
	}

	// Print the complete log line color-coded to stderr
	fmt.Fprintf(os.Stderr, colorFormat+"\n", fullLine)
}

// Package level functions delegating to defaultLogger

func INFO(v ...interface{})                  { defaultLogger.INFO(v...) }
func Info(v ...interface{})                  { defaultLogger.Info(v...) }
func Infof(format string, v ...interface{})   { defaultLogger.Infof(format, v...) }

func WARN(v ...interface{})                  { defaultLogger.WARN(v...) }
func Warn(v ...interface{})                  { defaultLogger.Warn(v...) }
func Warnf(format string, v ...interface{})   { defaultLogger.Warnf(format, v...) }

func DEBUG(v ...interface{})                 { defaultLogger.DEBUG(v...) }
func Debug(v ...interface{})                 { defaultLogger.Debug(v...) }
func Debugf(format string, v ...interface{})  { defaultLogger.Debugf(format, v...) }

func ERR(v ...interface{})                   { defaultLogger.ERR(v...) }
func Err(v ...interface{})                   { defaultLogger.Err(v...) }
func Errf(format string, v ...interface{})    { defaultLogger.Errf(format, v...) }

func FATAL(v ...interface{})                 { defaultLogger.FATAL(v...) }
func Fatal(v ...interface{})                 { defaultLogger.Fatal(v...) }
func Fatalf(format string, v ...interface{})  { defaultLogger.Fatalf(format, v...) }

// Logger methods

func (l *Logger) INFO(v ...interface{}) {
	l.logMsg(LevelInfo, "info", infoLogFile, infoLogPrefix, infoColor, v...)
}
func (l *Logger) Info(v ...interface{}) { l.INFO(v...) }
func (l *Logger) Infof(format string, v ...interface{}) {
	l.logMsgf(LevelInfo, "info", infoLogFile, infoLogPrefix, infoColor, format, v...)
}

func (l *Logger) WARN(v ...interface{}) {
	l.logMsg(LevelInfo, "warn", warnLogFile, warnLogPrefix, warnColor, v...)
}
func (l *Logger) Warn(v ...interface{}) { l.WARN(v...) }
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.logMsgf(LevelInfo, "warn", warnLogFile, warnLogPrefix, warnColor, format, v...)
}

func (l *Logger) DEBUG(v ...interface{}) {
	l.logMsg(LevelDebug, "debug", debugLogFile, debugLogPrefix, debugColor, v...)
}
func (l *Logger) Debug(v ...interface{}) { l.DEBUG(v...) }
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.logMsgf(LevelDebug, "debug", debugLogFile, debugLogPrefix, debugColor, format, v...)
}

func (l *Logger) ERR(v ...interface{}) {
	l.logMsg(LevelError, "error", errorLogFile, errorLogPrefix, errorColor, v...)
}
func (l *Logger) Err(v ...interface{}) { l.ERR(v...) }
func (l *Logger) Errf(format string, v ...interface{}) {
	l.logMsgf(LevelError, "error", errorLogFile, errorLogPrefix, errorColor, format, v...)
}

func (l *Logger) FATAL(v ...interface{}) {
	l.logMsg(LevelError, "fatal", fatalLogFile, fatalLogPrefix, errorColor, v...)
}
func (l *Logger) Fatal(v ...interface{}) { l.FATAL(v...) }
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logMsgf(LevelError, "fatal", fatalLogFile, fatalLogPrefix, errorColor, format, v...)
}
