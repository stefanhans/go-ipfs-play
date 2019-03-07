# Start IPFS Daemon via exec.Cmd

Inspired by a [forum post](https://discuss.ipfs.io/t/ipfs-start-ipfs-daemon-direct-from-the-go-application/4952) I've played around to start an IPFS daemon via `ipfs daemon`.


- created an extra IPFS directory 

```bash
export IPFS_PATH=~/.ipfs_daemon
ipfs init
```

- written `main.go`

- run it using the extra IPFS directory

```bash
export IPFS_PATH=~/.ipfs_daemon
go run main.go
```

- opened the WebUI 👍

- ask the API something 👍

```bash
curl -s "http://localhost:5001/api/v0/id" | jq
```

- run it again 

```bash
go run main.go
```
