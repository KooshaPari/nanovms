package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kooshapari/nanovms/internal/api"
	"github.com/kooshapari/nanovms/internal/listen"
	"github.com/kooshapari/nanovms/internal/token"
	"github.com/kooshapari/nanovms/pkg/deploy"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "deploy":
		deployCmd()
	case "serve":
		serveCmd(os.Args[2:])
	case "token":
		tokenCmd(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "nvms: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: nvms <command> [flags]

Commands:
  deploy              Deploy a workload to the specified tier
  serve               Start the NVMS daemon (HTTP over UDS)
  token               Manage bearer tokens (mint, list, remove)
  help                Show this help`)
}

func deployCmd() {
	deploySet := flag.NewFlagSet("deploy", flag.ExitOnError)
	tier := deploySet.Int("tier", 1, "Deployment tier (1=WASM, 2=gVisor, 3=Firecracker)")
	config := deploySet.String("config", "nvms.yaml", "Path to deployment config")
	_ = deploySet.Parse(os.Args[2:])

	ctx := context.Background()
	if err := deploy.Deploy(ctx, *tier, *config); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deployment completed successfully (tier=%d, config=%s)\n", *tier, *config)
}

func serveCmd(args []string) {
	serveSet := flag.NewFlagSet("serve", flag.ExitOnError)
	socketPath := serveSet.String("socket", "", "UDS socket path (default: $XDG_RUNTIME_DIR/nanovms/routed.sock)")
	tokenFile := serveSet.String("token-file", "", "Path to token file (default: $XDG_CONFIG_DIR/nanovms/tokens)")
	runBase := serveSet.String("run-base", "", "Runtime base dir (default: /run/user/<uid> or /tmp)")
	_ = serveSet.Parse(args)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Resolve defaults
	if *socketPath == "" {
		runDir := os.Getenv("XDG_RUNTIME_DIR")
		if runDir == "" {
			runDir = "/tmp"
		}
		*socketPath = runDir + "/nanovms/routed.sock"
	}
	if *tokenFile == "" {
		cfgDir := os.Getenv("XDG_CONFIG_DIR")
		if cfgDir == "" {
			cfgDir = "/etc/nanovms"
		}
		*tokenFile = cfgDir + "/tokens"
	}
	if *runBase == "" {
		*runBase = os.Getenv("XDG_RUNTIME_DIR")
		if *runBase == "" {
			*runBase = "/tmp"
		}
	}

	// Load tokens
	tm, err := token.NewManager(*tokenFile)
	if err != nil {
		log.Fatalf("token manager: %v", err)
	}

	// Create UDS listener
	ln, err := listen.NewUDS(ctx, *socketPath, *runBase)
	if err != nil {
		log.Fatalf("uds listener: %v", err)
	}
	defer ln.Close()

	// Start server
	log.Printf("NVMS daemon starting on %s (tokens from %s)", *socketPath, *tokenFile)
	if err := api.Serve(ctx, ln, api.Handlers{Token: tm}); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func tokenCmd(args []string) {
	tokenSet := flag.NewFlagSet("token", flag.ExitOnError)
	_ = tokenSet.Parse(args)

	if tokenSet.NArg() == 0 || tokenSet.Arg(0) == "help" {
		fmt.Println(`Usage: nvms token <subcommand>

Subcommands:
  mint                 Generate a new bearer token
  list                 List configured tokens
  remove <token>       Remove a token`)
		return
	}

	switch tokenSet.Arg(0) {
	case "mint":
		tok, err := token.MintToken()
		if err != nil {
			log.Fatalf("mint: %v", err)
		}
		fmt.Println(tok)
	default:
		fmt.Fprintf(os.Stderr, "nvms token: unknown subcommand %q\n", tokenSet.Arg(0))
		os.Exit(1)
	}
}
