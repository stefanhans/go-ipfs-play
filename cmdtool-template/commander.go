package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

var (
	commands    = make(map[string]string)
	commandKeys []string

	tmpDebugfile *os.File
)

func commandsInit() {
	commands = make(map[string]string)

	// Internals
	commands["log"] = "log (on <filename>)|off \n\t log starts or stops writing logging output in the specified file\n"

	commands["quit"] = "quit  \n\t close the session and exit\n"

	// Developer
	commands["play"] = "play  \n\t for developer playing\n"

	// To store the keys in sorted order
	for commandKey := range commands {
		commandKeys = append(commandKeys, commandKey)
	}
	sort.Strings(commandKeys)
}

// Execute a command specified by the argument string
func executeCommand(commandline string) bool {

	// Trim prefix and split string by white spaces
	commandFields := strings.Fields(commandline)

	// Check for empty string without prefix
	if len(commandFields) > 0 {

		// Switch according to the first word and call appropriate function with the rest as arguments
		switch commandFields[0] {

		case "log":
			cmdLogging(commandFields[1:])
			return true

		case "quit":
			quitCmdTool(commandFields[1:])
			return true

		case "play":
			play(commandFields[1:])
			return true

		default:
			usage()
			return false
		}
	}
	return false
}

// Display the usage of all available commands
func usage() {
	for _, key := range commandKeys {
		fmt.Printf("%v\n", commands[key])
	}

}

func quitCmdTool(arguments []string) {

	// Get rid of warnings
	_ = arguments

	os.Exit(0)
}

func play(arguments []string) {

	// Get rid of warnings
	_ = arguments

	log.Printf("CMD: play\n")
}

func cmdLogging(arguments []string) {

	if len(arguments) == 0 ||
		(len(arguments) == 1 && arguments[0] != "off") {
		fmt.Printf("Error: wrong input. Usage: \n\t 'log (on <filename>) | off\n")

		return
	}

	if arguments[0] == "on" && len(arguments) > 1 {
		log.Printf("Switch to logging by command to %q\n", arguments[1])
		tmpDebugfile, err = startLogging(arguments[1])
		if err != nil {
			fmt.Printf("Error: startLogging: %v\n", err)
		} else {
			log.Printf("Start logging by command to %q\n", arguments[1])
		}

		return
	}

	if arguments[0] == "off" {
		log.Printf("Stop logging by command")
		defer tmpDebugfile.Close()

		// Start debugging to file, if switched on or filename specified
		if *debug || len(*debugfilename) > 0 {

			if len(*debugfilename) == 0 {

				// Prepare logfile for logging
				year, month, day := time.Now().Date()
				hour, minute, second := time.Now().Clock()
				logfilename = fmt.Sprintf("cmdtool-%s-%v%02d%02d%02d%02d%02d.log", name,
					year, int(month), int(day), int(hour), int(minute), int(second))
			} else {
				logfilename = *debugfilename
			}
			log.Printf("Switch logging to %q\n", logfilename)
			_ = tmpDebugfile.Close()

			_, err := startLogging(logfilename)
			if err != nil {
				panic(err)
			}
			log.Printf("Switch back from logging by command to %q\n", tmpDebugfile.Name())
		}
	}
}
