package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	logfilename string
)

func startLogging(logname string) (*os.File, error) {

	// Config logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	//log.SetPrefix("DEBUG: ")

	if len(logname) == 0 {

		// Prepare logfile for logging
		year, month, day := time.Now().Date()
		hour, minute, second := time.Now().Clock()
		logfilename = fmt.Sprintf("cmdtool-%s-%v%02d%02d%02d%02d%02d.log", name,
			year, int(month), int(day), int(hour), int(minute), int(second))
	} else {
		logfilename = logname
	}
	logfile, err := os.OpenFile(logfilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening logfile %v: %v", logfilename, err)
	}

	// Switch logging to logfile
	log.SetOutput(logfile)

	return logfile, nil
}
