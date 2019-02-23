package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/peterh/liner"
)

var (
	name string

	err error

	debug         *bool
	debugfilename *string
)

func prompt() string {
	return fmt.Sprintf("< %s %s> ", time.Now().Format("Jan 2 15:04:05.000"), name)
}

func main() {

	// todo -nolog and refactor debug -> log

	// debug switches on debugging
	debug = flag.Bool("debug", true, "switches on debugging")

	// debugfilename is the file to write debugging output
	debugfilename = flag.String("debugfile", "", "file to write debugging output to, use /dev/null to suppress debugging")

	// Parse input and check arguments
	flag.Parse()
	if flag.NArg() < 1 {
		_, _ = fmt.Fprintln(os.Stderr, "missing or wrong parameter: <name>\n\n"+
			"Usage: ./cmdtool-tempate [-debug=false | -debugfile <logfilename>] <name>")
		os.Exit(1)
	}
	name = flag.Arg(0)

	// Start debugging to file, if switched on or filename specified
	if *debug || len(*debugfilename) > 0 {

		debugfile, err := startLogging(*debugfilename)
		if err != nil {
			panic(err)
		}
		defer debugfile.Close()

		// Current logfilename
		fmt.Printf("Start logging to %q\n", logfilename)

		// First entry in the logfile
		log.Printf("Session starting\n")
	}

	// Initialize commands
	commandsInit()

	// Start loop with history and completion
	err = interactiveLoop()
	if err != nil {
		panic(err)
	}
}

func interactiveLoop() error {
	s := liner.NewLiner()
	s.SetTabCompletionStyle(liner.TabPrints)
	s.SetCompleter(func(line string) (ret []string) {
		for _, c := range commandKeys {
			if strings.HasPrefix(c, line) {
				ret = append(ret, c)
			}
		}
		return
	})
	defer s.Close()
	for {
		p, err := s.Prompt(prompt())
		if err == io.EOF {
			return nil
		}
		if err != nil {
			panic(err)
		}
		if executeCommand(p) {
			s.AppendHistory(p)
		}
	}
}
