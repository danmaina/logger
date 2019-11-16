package logger

import (
	"log"
	"os"
)

var outputDir *string
var logLevel *int

// Create Log File Names
const (
	infoLogFile  = "INFO.log"
	debugLogFile = "DEBUG.log"
	traceLogFile = "TRACE.log"
	errorLogFile = "ERROR.log"
	fatalLogFile = "FATAL.log"

	infoLogPrefix  = "[INFO] || "
	errorLogPrefix = "[ERROR] || "
	debugLogPrefix = "[DEBUG] || "
	traceLogPrefix = "[TRACE] || "
	fatalLogPrefix = "[FATAL] || "
)

// Write info logs
func INFO(v ...interface{}) {

	if logLevel == nil || *logLevel > 1 {
		if outputDir != nil {
			err := logToFile(infoLogFile, infoLogPrefix, v)
			if err != nil {
				log.Println("Could not log to the info log file")
			}
		}

		log.Println(infoLogPrefix, v)
	}

}

// Write debug logs
func DEBUG(v ...interface{}) {
	if logLevel == nil || *logLevel > 2 {
		if outputDir != nil {
			err := logToFile(debugLogFile, debugLogPrefix, v)
			if err != nil {
				log.Println("Could not log to the debug log file")
			}
		}

		log.Println(debugLogPrefix, v)
	}
}

//write trace logs
func TRACE(v ...interface{}) {
	if logLevel == nil || *logLevel > 3 {
		if outputDir != nil {
			err := logToFile(traceLogFile, traceLogPrefix, v)
			if err != nil {
				log.Println("Could not log to the trace log file")
			}
		}

		log.Println(traceLogPrefix, v)
	}
}

// write error logs
func ERR(v ...interface{}) {
	if logLevel == nil || *logLevel > 0 {
		if outputDir != nil {
			err := logToFile(errorLogFile, errorLogPrefix, v)
			if err != nil {
				log.Println("Could not log to the error log file")
			}
		}

		log.Println(errorLogPrefix, v)
	}
}

// Write fatal logs
func FATAL(v ...interface{}) {
	if logLevel == nil || *logLevel == 0 {
		if outputDir != nil {
			err := logToFile(fatalLogFile, fatalLogPrefix, v)
			if err != nil {
				log.Println("Could not log to the fatal log file")
			}
		}

		log.Println(fatalLogPrefix, v)
	}
}

// Set the log output directory
func SetLogsDirectory(dir string) {

	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		log.Println("Could not set the log directory: ", err)
	}

	outputDir = &dir
}

// Set log level
func SetLogLevel(level int) {

	// Log level 0 is FATAL
	// Log level 1 is ERROR
	// Log level 2 is INFO
	// Log level 3 is DEBUG
	// log level 4 is TRACE

	if level > 5 {
		*logLevel = 1
	}

	logLevel = &level
}

// Set the correct log file for writing
func logToFile(file string, v ...interface{}) error {

	if *outputDir == "" {
		return nil
	}

	f, err := os.OpenFile(*outputDir+"/"+file, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)

	if err != nil {
		log.Println("Error while opening file: ", err)
		return err
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println(v...)
	return nil
}
