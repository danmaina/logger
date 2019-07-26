package logger

import (
	"log"
	"os"
)

var outputDir *string
var logPrefix *string

// Create Log File Names
const (
	infoLog = "INFO.log"
	debugLog = "DEBUG.log"
	traceLog = "TRACE.log"
	errorLog = "ERROR.log"
	fatalLog = "FATAL.log"
)

func INFO(v ...interface{}) {
	err := setLogFile(infoLog)

	if err != nil {
		log.Println("Could not log to the info log file")
	}

	log.Println(*logPrefix, "[INFO] :: ", v)
}

func DEBUG(v ...interface{}) {
	err := setLogFile(debugLog)

	if err != nil {
		log.Println("Could not log to the debug log file")
	}

	log.Println(*logPrefix, "[DEBUG] :: ", v)
}

func TRACE(v ...interface{}) {
	err := setLogFile(traceLog)

	if err != nil {
		log.Println("Could not log to the trace log file")
	}

	log.Println(*logPrefix, "[TRACE] :: ", v)
}

func ERR(v ...interface{}) {
	err := setLogFile(errorLog)

	if err != nil {
		log.Println("Could not log to the error log file")
	}

	log.Println(*logPrefix, "[ERROR] :: ", v)
}

func FATAL(v ...interface{}) {
	err := setLogFile(fatalLog)

	if err != nil {
		log.Println("Could not log to the fatal log file")
	}

	log.Println(*logPrefix, "[FATAL] :: ", v)
}

func SetLogsDirectory(dir string) {

	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		log.Println("Could not set the log directory: ", err)
	}

	outputDir = &dir
}

func SetLogPrefix(prefix string) {
	logPrefix = &prefix
}

// Set the correct log file for writing
func setLogFile(file string) error {

	if *outputDir == "" {
		return nil
	}

	f, err := os.OpenFile(*outputDir + "/" + file,  os.O_RDWR | os.O_CREATE | os.O_APPEND, os.ModePerm)

	if err != nil {
		log.Println("Error while opening file: ", err)
		return err
	}
	defer f.Close()

	log.SetOutput(f)
	return nil
}

