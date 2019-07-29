package logger

import (
	"log"
	"os"
)

var outputDir *string

// Create Log File Names
const (
	infoLog  = "INFO.log"
	debugLog = "DEBUG.log"
	traceLog = "TRACE.log"
	errorLog = "ERROR.log"
	fatalLog = "FATAL.log"
)

// Write info logs
func INFO(v ...interface{}) {
	if outputDir != nil {
		err := logToFile(infoLog, "[INFO] || ", v)
		if err != nil {
			log.Println("Could not log to the info log file")
		}
	}

	log.Println("[INFO] || ", v)
}

// Write debug logs
func DEBUG(v ...interface{}) {
	if outputDir != nil {
		err := logToFile(debugLog, "[DEBUG] || ", v)
		if err != nil {
			log.Println("Could not log to the debug log file")
		}
	}

	log.Println("[DEBUG] || ", v)
}

//write trace logs
func TRACE(v ...interface{}) {
	if outputDir != nil {
		err := logToFile(traceLog, "[TRACE] || ", v)
		if err != nil {
			log.Println("Could not log to the trace log file")
		}
	}

	log.Println("[TRACE] || ", v)
}

// write error logs
func ERR(v ...interface{}) {
	if outputDir != nil {
		err := logToFile(errorLog, "[ERROR] || ", v)
		if err != nil {
			log.Println("Could not log to the error log file")
		}
	}

	log.Println("[ERROR] || ", v)
}

// Write fatal logs
func FATAL(v ...interface{}) {
	if outputDir != nil {
		err := logToFile(fatalLog, " [FATAL] || ", v)
		if err != nil {
			log.Println("Could not log to the fatal log file")
		}
	}

	log.Println(" [FATAL] || ", v)
}

// Set the log output directory
func SetLogsDirectory(dir string) {

	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		log.Println("Could not set the log directory: ", err)
	}

	outputDir = &dir
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
