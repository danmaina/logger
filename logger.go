package logger

import (
	"fmt"
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
	info := formatLogString("[INFO] :: ", v)

	if outputDir != nil {
		err := logToFile(infoLog, info)
		if err != nil {
			log.Println("Could not log to the info log file")
		}
	}

	log.Println(info)
}

// Write debug logs
func DEBUG(v ...interface{}) {
	debug := formatLogString("[DEBUG] :: ", v)

	if outputDir != nil {
		err := logToFile(debugLog, debug)
		if err != nil {
			log.Println("Could not log to the debug log file")
		}
	}

	log.Println(debug)
}

//write trace logs
func TRACE(v ...interface{}) {
	trace := formatLogString("[TRACE] :: ", v)

	if outputDir != nil {
		err := logToFile(traceLog, trace)
		if err != nil {
			log.Println("Could not log to the trace log file")
		}
	}

	log.Println(trace)
}

// write error logs
func ERR(v ...interface{}) {
	errL := formatLogString("[ERROR] :: ", v)
	if outputDir != nil {
		err := logToFile(errorLog, errL)
		if err != nil {
			log.Println("Could not log to the error log file")
		}
	}

	log.Println(errL)
}

// Write fatal logs
func FATAL(v ...interface{}) {
	fatal := formatLogString(" [FATAL] || ", v)
	if outputDir != nil {
		err := logToFile(fatalLog, fatal)
		if err != nil {
			log.Println("Could not log to the fatal log file")
		}
	}

	log.Println(fatal)
}

func formatLogString(v ...interface{}) string {
	return fmt.Sprintf("%v", v...)
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
func logToFile(file string, v interface{}) error {

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
	log.Println(v)
	return nil
}
