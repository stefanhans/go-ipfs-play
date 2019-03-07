package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func main() {

	// API server running?
	resp, err := http.Get("http://127.0.0.1:5001/api/v0/version")
	if err != nil {

		// Start via "ipfs daemon"
		cmd := exec.Command("ipfs")
		cmd.Args = []string{"ipfs", "daemon"}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			panic(err)
		}

		// Wait until API server is responding
		for {
			resp, err = http.Get("http://127.0.0.1:5001/api/v0/version")
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			break
		}
	} else {
		fmt.Printf("ipfs daemon already running\n")
	}
	resp.Body.Close()
}
