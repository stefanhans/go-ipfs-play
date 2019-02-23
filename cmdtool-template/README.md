Here is the template for an interactive command line tool. It comes with these basic features

- command completion and history
- interactive logging
- individual function integration 

```
Usage: ./cmdtool-tempate [-debug=false | -debugfile <logfilename>] <name>
```

##### Hello World Command

- introduce the new function `helloworld` in `commander.go`

Add it to the map of commands
```go
func commandsInit() {
	commands = make(map[string]string)
	
	// Hello World
	commands["helloworld"] = "helloworld [text] \n\t helloworld is the obvious example for creating a new interactive comand\n"


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
```

Add a new case in the switch statement of `func executeCommand` in `commander.go`
```go
func executeCommand(commandline string) bool {

	// Trim prefix and split string by white spaces
	commandFields := strings.Fields(commandline)

	// Check for empty string without prefix
	if len(commandFields) > 0 {

		// Switch according to the first word and call appropriate function with the rest as arguments
		switch commandFields[0] {

		case "helloworld":
			cmdHelloWorld(commandFields[1:])
			return true

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
```

Implement the actual function in `commander.go`
```go
func cmdHelloWorld(arguments []string) {

	// Write to command line
	fmt.Printf("Hello World %s\n", strings.Join(arguments, " "))


	// Write to logfile
	log.Printf("Log message from cmdHelloWorld(%s)\n", strings.Join(arguments, " "))
}
```

Build and execute
```bash
go build && ./cmdtool-template alice
Start logging to "cmdtool-alice-20190223105320.log"
< Feb 23 10:53:20.380 alice> helloworld
Hello World
< Feb 23 10:53:25.335 alice> helloworld from me
Hello World from me
< Feb 23 10:53:33.571 alice> quit
```

Open the logfile
```bash
cat cmdtool-alice-20190223105320.log
2019/02/23 10:53:20 main.go:62: Session starting
2019/02/23 10:53:25 commander.go:93: Log message from cmdHelloWorld()
2019/02/23 10:53:33 commander.go:93: Log message from cmdHelloWorld(from me)
```

##### Interactive Logging

Log on, log off
```bash
./cmdtool-template alice
Start logging to "cmdtool-alice-20190223112004.log"
< Feb 23 11:20:04.892 alice> log on helloworld.log
< Feb 23 11:20:13.229 alice> helloworld to helloworld.log
Hello World to helloworld.log
< Feb 23 11:20:19.662 alice> log off
< Feb 23 11:20:26.956 alice> quit
```

See the logfiles
```bash
ls -1rt *log
cmdtool-alice-20190223112004.log
helloworld.log
cmdtool-alice-20190223112026.log
```

and its messages
```bash
for f in cmdtool-alice-20190223112004.log helloworld.log cmdtool-alice-20190223112026.log
[for]> do
[for]> echo $f
[for]> cat $f
[for]> echo
[for]> done
cmdtool-alice-20190223112004.log
2019/02/23 11:20:04 main.go:62: Session starting
2019/02/23 11:20:13 commander.go:122: Switch to logging by command to "helloworld.log"

helloworld.log
2019/02/23 11:20:13 commander.go:127: Start logging by command to "helloworld.log"
2019/02/23 11:20:19 commander.go:94: Log message from cmdHelloWorld(to helloworld.log)
2019/02/23 11:20:26 commander.go:134: Stop logging by command
2019/02/23 11:20:26 commander.go:150: Switch logging to "cmdtool-alice-20190223112026.log"

cmdtool-alice-20190223112026.log
2019/02/23 11:20:26 commander.go:157: Switch back from logging by command to "helloworld.log"
```

Logging from other packages used has to be adapted individually. Have a look at other packages of the repository.


