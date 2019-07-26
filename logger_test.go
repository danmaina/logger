package logger

import (
	"os"
	"testing"
)

func TestSetLogsDirectory(t *testing.T) {
	SetLogsDirectory("/tmp/testLogger/")

	f, err := os.Stat("/tmp/testLogger/")

	if err != nil {
		t.Error("Error While Fetching Created Directory: ", err)
	}

	if !f.IsDir() {
		t.Error("The Directory was not created properly: ", f.Name())
	}
}

func TestINFO(t *testing.T) {

}