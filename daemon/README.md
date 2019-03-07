# Starting an IPFS Daemon

Inspired by a [forum post](https://discuss.ipfs.io/t/ipfs-start-ipfs-daemon-direct-from-the-go-application/4952) I've played around to start an IPFS daemon programmatically.



- created an extra IPFS directory 

```bash
export IPFS_PATH=~/.ipfs_daemon
ipfs init
```

- analysed `github.com/ipfs/go-ipfs/cmd/ipfs`

- copied content of function `daemonFunc` from `daemon.go`, walked through, and changed it to a more rudimentary version

- run it using the extra IPFS directory

```bash
export IPFS_PATH=~/.ipfs_daemon
go run main.go
```

- opened the WebUI 👍

- ask the API something 👍

```bash
curl -s "http://localhost:5001/api/v0/bootstrap/list" | jq
```

So, it works and it was an educational experience. I'm not sure if it is the right way to use `go-ipfs`. 🤔  
