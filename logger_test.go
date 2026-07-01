package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestSetLogLevel(t *testing.T) {
	// Test levels and clamping
	SetLogLevel(-1)
	if defaultLogger.level != LevelError {
		t.Errorf("Expected LevelError (0), got %v", defaultLogger.level)
	}

	SetLogLevel(3)
	if defaultLogger.level != LevelDebug {
		t.Errorf("Expected LevelDebug (2), got %v", defaultLogger.level)
	}

	SetLogLevel(int(LevelInfo))
	if defaultLogger.level != LevelInfo {
		t.Errorf("Expected LevelInfo (1), got %v", defaultLogger.level)
	}
}

func TestSetLogsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "test_logs")

	SetLogsDirectory(logDir)

	if defaultLogger.outputDir != logDir {
		t.Errorf("Expected log directory %s, got %s", logDir, defaultLogger.outputDir)
	}

	// Verify that the directory was created
	fi, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("Failed to stat log directory: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("Expected a directory, but got file")
	}
}

func TestFileLogging(t *testing.T) {
	tempDir := t.TempDir()
	SetLogsDirectory(tempDir)
	SetLogLevel(int(LevelDebug)) // Enable all logs
	defer func() {
		// Reset defaultLogger to not log to file after this test
		defaultLogger.outputDir = ""
	}()

	testCases := []struct {
		name     string
		logFunc  func(v ...interface{})
		fileName string
		prefix   string
		content  string
	}{
		{"INFO", INFO, infoLogFile, infoLogPrefix, "info message"},
		{"WARN", WARN, warnLogFile, warnLogPrefix, "warn message"},
		{"DEBUG", DEBUG, debugLogFile, debugLogPrefix, "debug message"},
		{"ERR", ERR, errorLogFile, errorLogPrefix, "error message"},
		{"FATAL", FATAL, fatalLogFile, fatalLogPrefix, "fatal message"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write log
			tc.logFunc(tc.content)

			// Read file content
			filePath := filepath.Join(tempDir, tc.fileName)
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read log file %s: %v", tc.fileName, err)
			}

			contentStr := string(data)
			// Check prefix and content
			if !strings.Contains(contentStr, tc.prefix) {
				t.Errorf("Expected log file to contain prefix %q, but got: %q", tc.prefix, contentStr)
			}
			if !strings.Contains(contentStr, tc.content) {
				t.Errorf("Expected log file to contain content %q, but got: %q", tc.content, contentStr)
			}
			// Verify brackets are NOT present in the log output (formatting fix validation)
			bracketContent := "[" + tc.content + "]"
			if strings.Contains(contentStr, bracketContent) {
				t.Errorf("Log output contains raw slice brackets: %q", contentStr)
			}
		})
	}
}

func TestLogLevelThresholds(t *testing.T) {
	t.Run("LevelError Threshold", func(t *testing.T) {
		tempDir := t.TempDir()
		l := New(LevelError, tempDir)

		l.INFO("this info log should be ignored")
		l.WARN("this warn log should be ignored")
		l.DEBUG("this debug log should be ignored")
		l.ERR("this error log should be written")
		l.FATAL("this fatal log should be written")

		// Verify files
		if _, err := os.Stat(filepath.Join(tempDir, infoLogFile)); !os.IsNotExist(err) {
			t.Errorf("INFO log file should not exist")
		}
		if _, err := os.Stat(filepath.Join(tempDir, warnLogFile)); !os.IsNotExist(err) {
			t.Errorf("WARN log file should not exist")
		}
		if _, err := os.Stat(filepath.Join(tempDir, debugLogFile)); !os.IsNotExist(err) {
			t.Errorf("DEBUG log file should not exist")
		}

		errPath := filepath.Join(tempDir, errorLogFile)
		data, err := os.ReadFile(errPath)
		if err != nil {
			t.Fatalf("Failed to read ERROR log file: %v", err)
		}
		if !strings.Contains(string(data), "this error log should be written") {
			t.Errorf("ERROR log file missing content: %q", string(data))
		}

		fatalPath := filepath.Join(tempDir, fatalLogFile)
		data, err = os.ReadFile(fatalPath)
		if err != nil {
			t.Fatalf("Failed to read FATAL log file: %v", err)
		}
		if !strings.Contains(string(data), "this fatal log should be written") {
			t.Errorf("FATAL log file missing content: %q", string(data))
		}
	})

	t.Run("LevelInfo Threshold", func(t *testing.T) {
		tempDir := t.TempDir()
		l := New(LevelInfo, tempDir)

		l.INFO("this info log should be written")
		l.WARN("this warn log should be written")
		l.DEBUG("this debug log should be ignored")
		l.ERR("this error log should be written")
		l.FATAL("this fatal log should be written")

		// Verify debug file does not exist
		if _, err := os.Stat(filepath.Join(tempDir, debugLogFile)); !os.IsNotExist(err) {
			t.Errorf("DEBUG log file should not exist")
		}

		// Verify info, warn, err, fatal files
		for _, file := range []string{infoLogFile, warnLogFile, errorLogFile, fatalLogFile} {
			data, err := os.ReadFile(filepath.Join(tempDir, file))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", file, err)
			}
			if !strings.Contains(string(data), "should be written") {
				t.Errorf("%s missing expected log content: %q", file, string(data))
			}
		}
	})

	t.Run("LevelDebug Threshold", func(t *testing.T) {
		tempDir := t.TempDir()
		l := New(LevelDebug, tempDir)

		l.INFO("info msg")
		l.WARN("warn msg")
		l.DEBUG("debug msg")
		l.ERR("err msg")
		l.FATAL("fatal msg")

		// Verify all files exist and have content
		for _, file := range []string{infoLogFile, warnLogFile, debugLogFile, errorLogFile, fatalLogFile} {
			data, err := os.ReadFile(filepath.Join(tempDir, file))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", file, err)
			}
			if len(data) == 0 {
				t.Errorf("%s is empty", file)
			}
		}
	})
}

func TestFormattedLogging(t *testing.T) {
	tempDir := t.TempDir()
	l := New(LevelDebug, tempDir)

	l.Infof("info format %d %s", 123, "test")
	l.Warnf("warn format %s", "hello")
	l.Errf("error format %.2f", 3.14)
	l.Fatalf("fatal format %s", "boom")

	// Check info file
	infoPath := filepath.Join(tempDir, infoLogFile)
	data, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatalf("Failed to read INFO log file: %v", err)
	}
	if !strings.Contains(string(data), "info format 123 test") {
		t.Errorf("Expected formatted info log message, got: %q", string(data))
	}

	// Check warn file
	warnPath := filepath.Join(tempDir, warnLogFile)
	data, err = os.ReadFile(warnPath)
	if err != nil {
		t.Fatalf("Failed to read WARN log file: %v", err)
	}
	if !strings.Contains(string(data), "warn format hello") {
		t.Errorf("Expected formatted warn log message, got: %q", string(data))
	}

	// Check fatal file
	fatalPath := filepath.Join(tempDir, fatalLogFile)
	data, err = os.ReadFile(fatalPath)
	if err != nil {
		t.Fatalf("Failed to read FATAL log file: %v", err)
	}
	if !strings.Contains(string(data), "fatal format boom") {
		t.Errorf("Expected formatted fatal log message, got: %q", string(data))
	}
}

func TestConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	l := New(LevelDebug, tempDir)

	// Suppress standard log output printed to stderr for cleaner test output
	// while we run concurrency checks.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Drain the pipe concurrently in a background goroutine to avoid deadlocks
	// when the pipe buffer (typically 64KB on Linux) gets full.
	var buf bytes.Buffer
	var pipeWg sync.WaitGroup
	pipeWg.Add(1)
	go func() {
		defer pipeWg.Done()
		_, _ = io.Copy(&buf, r)
	}()

	defer func() {
		w.Close()
		r.Close()
		pipeWg.Wait()
		os.Stderr = oldStderr
	}()

	var wg sync.WaitGroup
	workers := 10
	logsPerWorker := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < logsPerWorker; j++ {
				l.Infof("worker %d log %d", workerID, j)
				l.Warnf("worker %d warn %d", workerID, j)
				l.Err("worker error log")
				l.Fatal("worker fatal log")
			}
		}(i)
	}

	wg.Wait()
	w.Close()
	pipeWg.Wait()

	// Check that log files were created and have content
	infoPath := filepath.Join(tempDir, infoLogFile)
	fi, err := os.Stat(infoPath)
	if err != nil {
		t.Fatalf("Failed to stat info log: %v", err)
	}
	if fi.Size() == 0 {
		t.Errorf("Expected info log to have content, got size 0")
	}
	if buf.Len() == 0 {
		t.Log("No logs captured on stderr, or pipe read completed")
	}
}

func TestLokiShipper(t *testing.T) {
	var mu sync.Mutex
	var receivedRequests []lokiPushRequest

	// Create test Loki server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/push" {
			t.Errorf("Expected path /loki/api/v1/push, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var req lokiPushRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			t.Errorf("Failed to unmarshal push request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedRequests = append(receivedRequests, req)
		mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	// Setup independent logger configured with Loki
	l := New(LevelDebug, "")
	l.SetLokiConfig(LokiConfig{
		URL:           ts.URL,
		Labels:        map[string]string{"env": "test", "job": "logger-test"},
		BatchWait:     50 * time.Millisecond,
		BatchCapacity: 5,
	})

	// Log some messages
	l.Info("Loki info message")
	l.Warn("Loki warn message")
	l.Err("Loki error message")
	l.Fatal("Loki fatal message")

	// Close logger to flush everything to Loki
	err := l.Close()
	if err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(receivedRequests) == 0 {
		t.Fatalf("No push requests received by test server")
	}

	// Verify entries were shipped
	foundInfo := false
	foundWarn := false
	foundErr := false
	foundFatal := false

	for _, req := range receivedRequests {
		for _, stream := range req.Streams {
			// Check labels
			if stream.Stream["env"] != "test" || stream.Stream["job"] != "logger-test" {
				t.Errorf("Incorrect stream labels: %v", stream.Stream)
			}
			for _, val := range stream.Values {
				line := val[1]
				if strings.Contains(line, "Loki info message") {
					foundInfo = true
				}
				if strings.Contains(line, "Loki warn message") {
					foundWarn = true
				}
				if strings.Contains(line, "Loki error message") {
					foundErr = true
				}
				if strings.Contains(line, "Loki fatal message") {
					foundFatal = true
				}
			}
		}
	}

	if !foundInfo || !foundWarn || !foundErr || !foundFatal {
		t.Errorf("Did not find all logged messages in Loki stream. Info=%t, Warn=%t, Err=%t, Fatal=%t", foundInfo, foundWarn, foundErr, foundFatal)
	}
}

func TestPrometheusMetrics(t *testing.T) {
	// Create Prometheus registry
	reg := prometheus.NewRegistry()

	l := New(LevelDebug, "")
	err := l.EnablePrometheus(reg)
	if err != nil {
		t.Fatalf("Failed to enable Prometheus: %v", err)
	}

	// Log messages
	l.Info("Log for Prometheus Info")
	l.Warn("Log for Prometheus Warn")
	l.Warn("Another Log for Prometheus Warn")
	l.Fatal("Log for Prometheus Fatal")

	// Gather metrics
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var foundCounter *dto.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "logger_messages_total" {
			foundCounter = mf
			break
		}
	}

	if foundCounter == nil {
		t.Fatalf("Did not find 'logger_messages_total' counter metric family")
	}

	infoCount := 0.0
	warnCount := 0.0
	fatalCount := 0.0

	for _, m := range foundCounter.GetMetric() {
		var levelLabel string
		for _, lp := range m.GetLabel() {
			if lp.GetName() == "level" {
				levelLabel = lp.GetValue()
			}
		}

		if levelLabel == "info" {
			infoCount = m.GetCounter().GetValue()
		} else if levelLabel == "warn" {
			warnCount = m.GetCounter().GetValue()
		} else if levelLabel == "fatal" {
			fatalCount = m.GetCounter().GetValue()
		}
	}

	if infoCount != 1.0 {
		t.Errorf("Expected info count to be 1.0, got %f", infoCount)
	}
	if warnCount != 2.0 {
		t.Errorf("Expected warn count to be 2.0, got %f", warnCount)
	}
	if fatalCount != 1.0 {
		t.Errorf("Expected fatal count to be 1.0, got %f", fatalCount)
	}
}

func TestSetTimeLayout(t *testing.T) {
	t.Run("Default Layout", func(t *testing.T) {
		tempDir := t.TempDir()
		l := New(LevelDebug, tempDir)
		l.Info("default layout msg")

		infoPath := filepath.Join(tempDir, infoLogFile)
		data, err := os.ReadFile(infoPath)
		if err != nil {
			t.Fatalf("Failed to read INFO log file: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, infoLogPrefix) {
			t.Errorf("Expected prefix %q, got: %q", infoLogPrefix, content)
		}
	})

	t.Run("Empty Layout (No Timestamp)", func(t *testing.T) {
		tempDir := t.TempDir()
		l := New(LevelDebug, tempDir)
		l.SetTimeLayout("")
		l.Info("no timestamp msg")

		infoPath := filepath.Join(tempDir, infoLogFile)
		data, err := os.ReadFile(infoPath)
		if err != nil {
			t.Fatalf("Failed to read INFO log file: %v", err)
		}

		expected := infoLogPrefix + "no timestamp msg\n"
		if string(data) != expected {
			t.Errorf("Expected file content to be exact %q, but got %q", expected, string(data))
		}
	})

	t.Run("Custom Layout (RFC3339)", func(t *testing.T) {
		tempDir := t.TempDir()
		l := New(LevelDebug, tempDir)
		l.SetTimeLayout(time.RFC3339)
		l.Info("rfc3339 msg")

		infoPath := filepath.Join(tempDir, infoLogFile)
		data, err := os.ReadFile(infoPath)
		if err != nil {
			t.Fatalf("Failed to read INFO log file: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "-") || !strings.Contains(content, "T") {
			t.Errorf("Expected content to contain rfc3339 format, got %q", content)
		}
	})
}
