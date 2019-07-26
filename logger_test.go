package logger

import (
	"os"
	"testing"
)

const (
	fileDir = "/tmp/testLogger/"
	logData = "test log Data"
)

func TestMain(m *testing.M) {

	code := m.Run()

	deleteLogFiles()

	os.Exit(code)
}

func deleteLogFiles() {
	_ = os.Remove(fileDir + infoLog)
	_ = os.Remove(fileDir + debugLog)
	_ = os.Remove(fileDir + traceLog)
	_ = os.Remove(fileDir + errorLog)
	_ = os.Remove(fileDir + fatalLog)
}

// Test that the logs output directory is being created properly
func TestSetLogsDirectory(t *testing.T) {
	SetLogsDirectory(fileDir)

	f, err := os.Stat(fileDir)

	if err != nil {
		t.Error("Error While Fetching Created Directory: ", err)
	}

	if !f.IsDir() {
		t.Error("The Directory was not created properly: ", f.Name())
	}
}

func TestINFO(t *testing.T) {

	// Create File Output Directory
	SetLogsDirectory(fileDir)

	// write a log using INFO
	INFO(logData)

	// Check the directory for the file
	f, err := os.Stat(fileDir + infoLog)

	if err != nil {
		t.Error("Error Finding Info log: ", err)
	}

	// Check that the file has content
	if f.Size() == 0 {
		t.Error("Log content was not written to file")
	}
}

func TestDEBUG(t *testing.T) {
	// Create File Output Directory
	SetLogsDirectory(fileDir)

	// write a log using INFO
	DEBUG(logData)

	// Check the directory for the file
	f, err := os.Stat(fileDir + debugLog)

	if err != nil {
		t.Error("Error Finding Info log: ", err)
	}

	// Check that the file has content
	if f.Size() == 0 {
		t.Error("Log content was not written to file")
	}
}

func TestTRACE(t *testing.T) {
	// Create File Output Directory
	SetLogsDirectory(fileDir)

	// write a log using INFO
	TRACE(logData)

	// Check the directory for the file
	f, err := os.Stat(fileDir + traceLog)

	if err != nil {
		t.Error("Error Finding Trace log: ", err)
	}

	// Check that the file has content
	if f.Size() == 0 {
		t.Error("Log content was not written to file")
	}
}

func TestERR(t *testing.T) {
	// Create File Output Directory
	SetLogsDirectory(fileDir)

	// write a log using INFO
	ERR(logData)

	// Check the directory for the file
	f, err := os.Stat(fileDir + errorLog)

	if err != nil {
		t.Error("Error Finding Error log: ", err)
	}

	// Check that the file has content
	if f.Size() == 0 {
		t.Error("Log content was not written to file")
	}
}

func TestFATAL(t *testing.T) {
	// Create File Output Directory
	SetLogsDirectory(fileDir)

	// write a log using FATAL
	FATAL(logData)

	// Check the directory for the file
	f, err := os.Stat(fileDir + fatalLog)

	if err != nil {
		t.Error("Error Finding Fatal log: ", err)
	}

	// Check that the file has content
	if f.Size() == 0 {
		t.Error("Log content was not written to file")
	}
}
