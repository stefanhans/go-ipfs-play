package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ipfs/go-ipfs-api"
)

var (
	commands    = make(map[string]string)
	commandKeys []string

	tmpDebugfile *os.File
)

func commandsInit() {
	commands = make(map[string]string)

	// Shell Exec
	commands["commands"] = "commands  \n\t commands shows all commands\n"

	// Commander
	commands["log"] = "log (on <filename>)|off \n\t log starts or stops writing logging output in the specified file\n"
	commands["quit"] = "quit  \n\t close the session and exit\n"

	// Scripting
	commands["execute"] = "execute file \n\t execute execute the commands in the file line by line, '#' is comment\n"
	commands["sleep"] = "sleep seconds \n\t sleep sleeps for seconds\n"
	commands["echo"] = "echo text_w/o_linebreak \n\t echo prints rest of line\n"

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

		case "commands":
			jsonCommands(commandFields[1:])
			return true

		case "log":
			cmdLogging(commandFields[1:])
			return true

		case "quit":
			quitCmdTool(commandFields[1:])
			return true

		case "execute":
			executeScript(commandFields[1:])
			return true

		case "sleep":
			sleepScript(commandFields[1:])
			return true

		case "echo":
			echoScript(commandFields[1:])
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

func jsonCommands(arguments []string) {

	sh := shell.NewShell("localhost:5001")

	var commands map[string]interface{}
	err = sh.Request("commands", "flags=true").Exec(context.Background(), &commands)
	if err != nil {
		fmt.Printf("commands.Exec(): %v\n", err)
	}
	//fmt.Printf("commands: %v\n", commands)

	jsonBytes, err := json.MarshalIndent(commands, "", "    ")
	if err != nil {
		fmt.Printf("json.MarshalIndent(): %v\n", err)
	}
	fmt.Printf("commands: %v\n", string(jsonBytes))
}

func quitCmdTool(arguments []string) {

	// Get rid of warnings
	_ = arguments

	os.Exit(0)
}

func scriptPrompt(scriptname string) string {
	return fmt.Sprintf("<%s %q> ", time.Now().Format("Jan 2 15:04:05.000"), scriptname)
}

func executeScript(arguments []string) {

	if len(arguments) == 0 {
		fmt.Printf("error: no filename to execute specified\n")
		return
	}

	b, err := ioutil.ReadFile(arguments[0])
	if err != nil {
		fmt.Printf("ioutil.ReadFile: %v\n", err)
		return
	}

	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" ||
			strings.Split(strings.TrimSpace(line), "")[0] == "#" {
			continue
		}
		echoScript(strings.Split(scriptPrompt(arguments[0])+line, " "))
		if _, ok := commands[strings.Split(line, " ")[0]]; ok {
			executeCommand(line)
		} else {
			fmt.Printf("error: %q is an unknown command\n", strings.Split(line, " ")[0])
		}
	}
}

func sleepScript(arguments []string) {

	var numSeconds int

	if len(arguments) == 0 {
		numSeconds = 1
	} else {
		numSeconds, err = strconv.Atoi(arguments[0])
	}

	time.Sleep(time.Second * time.Duration(numSeconds))
}

func echoScript(arguments []string) {

	fmt.Printf("%s\n", strings.Join(arguments, " "))
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

func play(arguments []string) {

	// Get rid of warnings
	_ = arguments

	log.Printf("CMD: play\n")

	sh := shell.NewShell("localhost:5001")

	var commands map[string]interface{}
	err = sh.Request("commands", "flags=true").Exec(context.Background(), &commands)
	if err != nil {
		fmt.Printf("commands.Exec(): %v\n", err)
	}
	//fmt.Printf("commands: %v\n", commands)

	jsonBytes, err := json.MarshalIndent(commands, "", "    ")
	if err != nil {
		fmt.Printf("json.MarshalIndent(): %v\n", err)
	}
	//fmt.Printf("commands: %v\n", string(jsonBytes))

	// Manually read from "unknown" JSON data
	var f interface{}
	err = json.Unmarshal(jsonBytes, &f)
	if err != nil {
		fmt.Printf("json.Unmarshal(b, conf): %v\n", err)
		return
	}
	m := f.(map[string]interface{})

	for k, v := range m {
		fmt.Println("\n")
		switch vv := v.(type) {
		case string:

			fmt.Printf("%q: %v\n", k, vv)

		case []interface{}:
			fmt.Println(k, "is an array:")
			for i, u := range vv {
				fmt.Println(i, u)
			}

		case map[string]interface{}:
			fmt.Printf("%q leads deeper via another map[string]interface{}\n", k)

			n := v.(map[string]interface{})
			fmt.Printf("%v\n", n)
		case nil:
			fmt.Printf("%q was not set in this configuration\n", k)
		default:
			fmt.Printf("%q is of a type %v\n", k, v)
		}
	}

	//bootstrapServer, err := sh.BootstrapAddDefault()
	//if err != nil {
	//	fmt.Printf("BootstrapAddDefault(): %v\n", err)
	//}
	//for i, b := range bootstrapServer {
	//
	//	fmt.Printf("bootstrapServer %d: %v\n", i, b)
	//}

	//var listOutput shell.PeersList
	//err = sh.Request("bootstrap/list").Exec(context.Background(), &listOutput)
	//if err != nil {
	//	fmt.Printf("bootstrap/list.Exec(): %v\n", err)
	//}
	//fmt.Printf("listOutput: %v\n", listOutput)

	//for i, b := range listOutput.Peers {
	//
	//	fmt.Printf("listOutput.Peers %d: %v\n", i, b)
	//}
}
