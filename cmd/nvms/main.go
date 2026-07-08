package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/kooshapari/nanovms/internal/adapters"
	"github.com/kooshapari/nanovms/internal/api"
	"github.com/kooshapari/nanovms/internal/listen"
	"github.com/kooshapari/nanovms/internal/sandbox"
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
	case "vm":
		vmCmd(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "nvms: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}


func vmCmd(args []string) {
	vmSet := flag.NewFlagSet("vm", flag.ExitOnError)
	socketPath := vmSet.String("socket", "", "UDS socket path")
	_ = vmSet.Parse(args)

	if vmSet.NArg() == 0 || vmSet.Arg(0) == "help" {
		fmt.Println(`Usage: nvms vm [--socket <path>] <subcommand> [args]

Subcommands:
  list                   List all sandboxes
  exec <id> <cmd...>     Execute a command in a sandbox
  logs <id>              Stream logs from a sandbox
  port-forward <id> <local-port>:<remote-port>  Create a port-forward tunnel`)
		return
	}

	cl := sandbox.NewClient(*socketPath)

	switch vmSet.Arg(0) {
	case "list":
		sandboxes, err := cl.ListSandboxes(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		for _, sb := range sandboxes {
			fmt.Printf("%s\t%s\t%s\n", sb.ID, sb.Name, sb.Status)
		}
	case "exec":
		if vmSet.NArg() < 3 {
			log.Fatal("usage: nvms vm exec <id> <cmd...>")
		}
		id := vmSet.Arg(1)
		cmd := vmSet.Args()[2:]
		out, err := cl.Exec(context.Background(), id, cmd)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
		if _, err := io.Copy(os.Stdout, out); err != nil {
			log.Fatal(err)
		}
	case "logs":
		if vmSet.NArg() < 2 {
			log.Fatal("usage: nvms vm logs <id>")
		}
		id := vmSet.Arg(1)
		out, err := cl.Logs(context.Background(), id, false)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
		if _, err := io.Copy(os.Stdout, out); err != nil {
			log.Fatal(err)
		}
	case "port-forward":
		if vmSet.NArg() != 3 {
			log.Fatal("usage: nvms vm port-forward <id> <local-port>:<remote-port>")
		}
		id := vmSet.Arg(1)
		ports := strings.SplitN(vmSet.Arg(2), ":", 2)
		if len(ports) != 2 {
			log.Fatal("invalid port spec, use <local>:<remote>")
		}
		local, err := strconv.Atoi(ports[0])
		if err != nil {
			log.Fatalf("invalid local port: %v", err)
		}
		remote, err := strconv.Atoi(ports[1])
		if err != nil {
			log.Fatalf("invalid remote port: %v", err)
		}
		addr, err := cl.PortForward(context.Background(), id, local, remote)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Port-forward active: %s\n", addr)
	default:
		fmt.Fprintf(os.Stderr, "nvms vm: unknown subcommand %q\n", vmSet.Arg(0))
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: nvms <command> [flags]

Commands:
  deploy              Deploy a workload to the specified tier
  serve               Start the NVMS daemon (HTTP over UDS)
  token               Manage bearer tokens (mint, list, remove)
  vm                  Manage VMs/sandboxes (exec, logs, port-forward)
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
	tier := serveSet.Int("tier", 3, "Sandbox tier (1=WASM, 2=gVisor, 3=Firecracker)")
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

	// Create sandbox adapter for the requested tier
	adapter, err := adapters.NewSandboxPort(*tier)
	if err != nil {
		log.Fatalf("adapter (tier %d): %v", *tier, err)
	}
	log.Printf("Using sandbox adapter: tier=%d", *tier)

	// Phase 2: audit logger (writes to nvms-audit.jsonl in temp dir)
	auditDir := os.TempDir()
	if p := os.Getenv("NVMS_AUDIT_PATH"); p != "" {
		auditDir = p
	}
	auditLog := api.NewAuditLogger(auditDir)

	// Start server
	log.Printf("NVMS daemon starting on %s (tokens from %s)", *socketPath, *tokenFile)
	if err := api.Serve(ctx, ln, api.Handlers{
		Port:     adapter,
		Token:    tm,
		AuditLog: auditLog,
	}); err != nil {
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
	case "list":
		fmt.Println("token list: not yet implemented (manager does not expose enumeration)",
			" — use 'nvms token mint' to generate a new token,",
			" check: cat /etc/nanovms/tokens",
		)
	default:
		fmt.Fprintf(os.Stderr, "nvms token: unknown subcommand %q\n", tokenSet.Arg(0))
		os.Exit(1)
	}
}
