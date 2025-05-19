package logger

import (
	"io"
	"log"
	"os"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func Init(logPath string) error {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	multiOut := io.MultiWriter(file, os.Stdout)

	InfoLogger = log.New(multiOut, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(multiOut, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}
