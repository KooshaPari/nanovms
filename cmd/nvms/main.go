package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kooshapari/nanovms/internal/api"
	"github.com/kooshapari/nanovms/internal/listen"
	"github.com/kooshapari/nanovms/internal/ports"
	"github.com/kooshapari/nanovms/internal/token"
	"github.com/kooshapari/nanovms/pkg/deploy"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "serve":
		serveCmd(os.Args[2:])
	case "token":
		tokenCmd(os.Args[2:])
	case "deploy":
		deployCmd(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: nvms <command> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  serve     Start the NVMS daemon on a Unix socket")
	fmt.Println("  token     Mint or rotate bearer tokens")
	fmt.Println("  deploy    Deploy a workload to the specified tier")
}

// serveCmd starts the daemon. Phase 1b stub: connects a nil-port router so
// the daemon is up and answering /healthz. The real SandboxPort integration
// arrives in Phase 1b.2 alongside the YAML config loader.
func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	socket := fs.String("socket", "/tmp/omniroute/nvms.sock", "UDS path")
	tokenFile := fs.String("tokens", "/etc/nanovms/tokens", "Bearer-token file")
	logFile := fs.String("log", "", "Optional log file (default stderr)")
	_ = fs.Parse(args)

	logger := log.New(os.Stderr, "nvms ", log.LstdFlags|log.Lmicroseconds)
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
		if err != nil {
			logger.Fatalf("open log: %v", err)
		}
		logger = log.New(f, "nvms ", log.LstdFlags)
	}

	tm, err := token.NewManager(*tokenFile)
	if err != nil {
		logger.Fatalf("token load: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := listen.NewUDS(ctx, *socket, "/tmp")
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Phase 1b stub port: a nil port rejects deploy calls with 501.
	var sandboxPort ports.SandboxPort
	_ = sandboxPort // wired in Phase 1b.2

	logger.Printf("serving on %s", *socket)
	if err := api.Serve(ctx, ln, api.Handlers{Port: nil, Token: tm}); err != nil {
		logger.Printf("serve: %v", err)
		os.Exit(1)
	}
}

// tokenCmd dispatches mint / list / rotate. Phase 1b stub: only mint.
func tokenCmd(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: nvms token <mint|list>")
		os.Exit(1)
	}
	switch args[0] {
	case "mint":
		tok, err := token.MintToken()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(tok)
	default:
		fmt.Printf("Unknown token command: %s\n", args[0])
		os.Exit(1)
	}
}

func deployCmd(args []string) {
	deploySet := flag.NewFlagSet("deploy", flag.ExitOnError)
	tier := deploySet.Int("tier", 1, "Deployment tier (1=WASM, 2=gVisor, 3=Firecracker)")
	config := deploySet.String("config", "nvms.yaml", "Path to deployment config")
	_ = deploySet.Parse(args)

	ctx := context.Background()
	if err := deploy.Deploy(ctx, *tier, *config); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deployment completed successfully (tier=%d, config=%s)\n", *tier, *config)
}
