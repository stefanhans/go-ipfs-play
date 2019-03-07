package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"

	oldcmds "github.com/ipfs/go-ipfs/commands"
	core "github.com/ipfs/go-ipfs/core"
	commands "github.com/ipfs/go-ipfs/core/commands"
	coreapi "github.com/ipfs/go-ipfs/core/coreapi"
	corehttp "github.com/ipfs/go-ipfs/core/corehttp"
	loader "github.com/ipfs/go-ipfs/plugin/loader"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"

	cmds "gx/ipfs/QmQkW9fnCsg9SLHdViiAh6qfBppodsPZVpU92dZLqYtEfs/go-ipfs-cmds"
	goprocess "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	ma "gx/ipfs/QmTZBfrPJmjWsCvHEtX5FE6KimVJhsJg5sBbqEFYf4UZtL/go-multiaddr"
	config "gx/ipfs/QmUAuYuiafnJRZxDDX7MuruMNsicYNuyub5vUeAcupUBNs/go-ipfs-config"
	manet "gx/ipfs/Qmc85NSvmSG4Frn9Vb2cBc1rMyULH6D3TNVEfCzSKoUpip/go-multiaddr-net"
)

func loadConfig(path string) (*config.Config, error) {
	return fsrepo.ConfigAt(path)
}

func loadPlugins(repoPath string) (*loader.PluginLoader, error) {
	pluginpath := filepath.Join(repoPath, "plugins")

	// check if repo is accessible before loading plugins
	var plugins *loader.PluginLoader
	ok, err := checkPermissions(repoPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		pluginpath = ""
	}
	plugins, err = loader.NewPluginLoader(pluginpath)
	if err != nil {
		fmt.Printf("loader.NewPluginLoader: %v\n", err)
	}

	if err := plugins.Initialize(); err != nil {
		fmt.Printf("plugins.Initialize: %v\n", err)
	}

	if err := plugins.Inject(); err != nil {
		fmt.Printf("plugins.Inject: %v\n", err)
	}
	return plugins, nil
}

func checkPermissions(path string) (bool, error) {
	_, err := os.Open(path)
	if os.IsNotExist(err) {
		// repo does not exist yet - don't load plugins, but also don't fail
		return false, nil
	}
	if os.IsPermission(err) {
		// repo is not accessible. error out.
		return false, fmt.Errorf("error opening repository at %s: permission denied", path)
	}

	return true, nil
}

// printSwarmAddrs prints the addresses of the host
func printSwarmAddrs(node *core.IpfsNode) {
	if !node.OnlineMode() {
		fmt.Println("Swarm not listening, running in offline mode.")
		return
	}

	var lisAddrs []string
	ifaceAddrs, err := node.PeerHost.Network().InterfaceListenAddresses()
	if err != nil {
		fmt.Printf("plugins.Inject: %v\n", err)
	}
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Sort(sort.StringSlice(lisAddrs))
	for _, addr := range lisAddrs {
		fmt.Printf("Swarm listening on %s\n", addr)
	}

	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Sort(sort.StringSlice(addrs))
	for _, addr := range addrs {
		fmt.Printf("Swarm announcing %s\n", addr)
	}

}

// serveHTTPApi collects options, creates listener, prints status message and starts serving requests
func serveHTTPApi(req *cmds.Request, cctx *oldcmds.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: GetConfig() failed: %s", err)
	}

	apiAddrs := make([]string, 0, 2)
	apiAddr, _ := req.Options[commands.ApiOption].(string)
	if apiAddr == "" {
		apiAddrs = cfg.Addresses.API
	} else {
		apiAddrs = append(apiAddrs, apiAddr)
	}

	listeners := make([]manet.Listener, 0, len(apiAddrs))
	for _, addr := range apiAddrs {
		apiMaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPApi: invalid API address: %q (err: %s)", apiAddr, err)
		}

		apiLis, err := manet.Listen(apiMaddr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPApi: manet.Listen(%s) failed: %s", apiMaddr, err)
		}

		// we might have listened to /tcp/0 - lets see what we are listing on
		apiMaddr = apiLis.Multiaddr()
		fmt.Printf("API server listening on %s\n", apiMaddr)
		fmt.Printf("WebUI: http://%s/webui\n", apiLis.Addr())
		listeners = append(listeners, apiLis)
	}

	gatewayOpt := corehttp.GatewayOption(false, corehttp.WebUIPaths...)

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("api"),
		corehttp.CheckVersionOption(),
		corehttp.CommandsOption(*cctx),
		corehttp.WebUIOption,
		gatewayOpt,
		corehttp.VersionOption(),
		defaultMux("/debug/vars"),
		defaultMux("/debug/pprof/"),
		corehttp.MutexFractionOption("/debug/pprof-mutex/"),
		corehttp.MetricsScrapingOption("/debug/metrics/prometheus"),
		corehttp.LogOption(),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: ConstructNode() failed: %s", err)
	}

	if err := node.Repo.SetAPIAddr(listeners[0].Multiaddr()); err != nil {
		return nil, fmt.Errorf("serveHTTPApi: SetAPIAddr() failed: %s", err)
	}

	errc := make(chan error)
	var wg sync.WaitGroup
	for _, apiLis := range listeners {
		wg.Add(1)
		go func(lis manet.Listener) {
			defer wg.Done()
			errc <- corehttp.Serve(node, manet.NetListener(lis), opts...)
		}(apiLis)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	return errc, nil
}

// defaultMux tells mux to serve path using the default muxer. This is
// mostly useful to hook up things that register in the default muxer,
// and don't provide a convenient http.Handler entry point, such as
// expvar and http/pprof.
func defaultMux(path string) corehttp.ServeOption {
	return func(node *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.Handle(path, http.DefaultServeMux)
		return mux, nil
	}
}

// serveHTTPGateway collects options, creates listener, prints status message and starts serving requests
func serveHTTPGateway(req *cmds.Request, cctx *oldcmds.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: GetConfig() failed: %s", err)
	}

	writable := cfg.Gateway.Writable

	gatewayAddrs := cfg.Addresses.Gateway
	listeners := make([]manet.Listener, 0, len(gatewayAddrs))
	for _, addr := range gatewayAddrs {
		gatewayMaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPGateway: invalid gateway address: %q (err: %s)", addr, err)
		}

		gwLis, err := manet.Listen(gatewayMaddr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
		}
		// we might have listened to /tcp/0 - lets see what we are listing on
		gatewayMaddr = gwLis.Multiaddr()

		if writable {
			fmt.Printf("Gateway (writable) server listening on %s\n", gatewayMaddr)
		} else {
			fmt.Printf("Gateway (readonly) server listening on %s\n", gatewayMaddr)
		}

		listeners = append(listeners, gwLis)
	}

	cmdctx := *cctx
	cmdctx.Gateway = true

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(writable, "/ipfs", "/ipns"),
		corehttp.VersionOption(),
		corehttp.CheckVersionOption(),
		corehttp.CommandsROOption(cmdctx),
	}

	if cfg.Experimental.P2pHttpProxy {
		opts = append(opts, corehttp.ProxyOption())
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPGateway: ConstructNode() failed: %s", err)
	}

	errc := make(chan error)
	var wg sync.WaitGroup
	for _, lis := range listeners {
		wg.Add(1)
		go func(lis manet.Listener) {
			defer wg.Done()
			errc <- corehttp.Serve(node, manet.NetListener(lis), opts...)
		}(lis)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	return errc, nil
}

// merge does fan-in of multiple read-only error channels
// taken from http://blog.golang.org/pipelines
func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	for _, c := range cs {
		if c != nil {
			wg.Add(1)
			go output(c)
		}
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {

	repoPath, err := fsrepo.BestKnownPath()
	if err != nil {
		return
	}

	//func(ctx context.Context, req *cmds.Request)

	ctx := context.Background()

	req := &cmds.Request{Context: ctx}

	plugins, err := loadPlugins(repoPath)
	if err != nil {
		fmt.Printf("loadPlugins: %v\n", err)
		return
	}

	cctx := oldcmds.Context{
		ConfigRoot: repoPath,
		LoadConfig: loadConfig,
		ReqLog:     &oldcmds.ReqLog{},
		Plugins:    plugins,
		ConstructNode: func() (n *core.IpfsNode, err error) {
			if req == nil {
				return nil, errors.New("constructing node without a request")
			}

			r, err := fsrepo.Open(repoPath)
			if err != nil { // repo is owned by the node
				return nil, err
			}

			// ok everything is good. set it on the invocation (for ownership)
			// and return it.
			n, err = core.NewNode(ctx, &core.BuildCfg{
				Repo: r,
			})
			if err != nil {
				return nil, err
			}

			n.SetLocal(true)
			return n, nil
		},
	}
	//fmt.Printf("cctx: %#v\n", cctx)

	repo, err := fsrepo.Open(cctx.ConfigRoot)
	if err != nil {
		fmt.Printf("fsrepo.Open: %v\n", err)
		return
	}
	//fmt.Printf("repo: %v\n", repo)

	cfg, err := cctx.GetConfig()
	if err != nil {
		fmt.Printf("cctx.GetConfig: %v\n", err)
		return
	}
	//fmt.Printf("cfg: %v\n", cfg)

	// Start assembling node config
	ncfg := &core.BuildCfg{
		Repo:                        repo,
		Permanent:                   true, // It is temporary way to signify that node is permanent
		Online:                      true,
		DisableEncryptedConnections: false,
	}
	//fmt.Printf("ncfg: %v\n", ncfg)

	cfg, err = repo.Config()
	if err != nil {
		fmt.Printf("repo.Config: %v\n", err)
		return
	}
	//fmt.Printf("cfg: %v\n", cfg)
	//fmt.Printf("cfg.Routing.Type: %v\n", cfg.Routing.Type)

	ncfg.Routing = core.DHTOption

	node, err := core.NewNode(req.Context, ncfg)
	if err != nil {
		fmt.Printf("core.NewNode: %v\n", err)
		return
	}
	node.SetLocal(false)

	printSwarmAddrs(node)

	defer func() {
		// We wait for the node to close first, as the node has children
		// that it will wait for before closing, such as the API server.
		node.Close()

		select {
		case <-req.Context.Done():
			fmt.Printf("Gracefully shut down daemon\n")
		default:
		}
	}()

	cctx.ConstructNode = func() (*core.IpfsNode, error) {
		return node, nil
	}

	// Start "core" plugins. We want to do this *before* starting the HTTP
	// API as the user may be relying on these plugins.
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		fmt.Printf("coreapi.NewCoreAPI: %v\n", err)
		return
	}
	err = cctx.Plugins.Start(api)
	if err != nil {
		fmt.Printf("cctx.Plugins.Start: %v\n", err)
		return
	}
	node.Process().AddChild(goprocess.WithTeardown(cctx.Plugins.Close))

	// construct api endpoint - every time
	apiErrc, err := serveHTTPApi(req, &cctx)
	if err != nil {
		fmt.Printf("serveHTTPApi: %v\n", err)
		return
	}

	// construct http gateway - if it is set in the config
	var gwErrc <-chan error
	if len(cfg.Addresses.Gateway) > 0 {
		var err error
		gwErrc, err = serveHTTPGateway(req, &cctx)
		if err != nil {
			fmt.Printf("serveHTTPGateway: %v\n", err)
			return
		}
	}

	// The daemon is *finally* ready.
	fmt.Printf("Daemon is ready\n")

	// Give the user some immediate feedback when they hit C-c
	go func() {
		<-req.Context.Done()
		fmt.Println("Received interrupt signal, shutting down...")
		fmt.Println("(Hit ctrl-c again to force-shutdown the daemon.)")
	}()

	// collect long-running errors and block for shutdown
	for err := range merge(apiErrc, gwErrc) {
		if err != nil {
			fmt.Printf("merge: %v\n", err)
			return
		}
	}
}
