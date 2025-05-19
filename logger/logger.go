package logger

import (
	"log"
	"os"
)

var (
	LogFile     *os.File
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func Init(logPath string) {
	var err error
	LogFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
