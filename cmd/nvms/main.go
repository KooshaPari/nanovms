package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters"
	"github.com/kooshapari/nanovms/internal/api"
	"github.com/kooshapari/nanovms/internal/listen"
	"github.com/kooshapari/nanovms/internal/sandbox"
	"github.com/kooshapari/nanovms/internal/token"
	"github.com/kooshapari/nanovms/pkg/deploy"
	"github.com/kooshapari/nanovms/pkg/environment"
	"github.com/kooshapari/nanovms/pkg/gpu"
	"github.com/kooshapari/nanovms/pkg/orchestrate"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

const (
	exitOK            = orchestrate.ExitOK
	exitUsage         = orchestrate.ExitUsage
	exitInvalidJSON   = orchestrate.ExitInvalidJSON
	exitEncodeFailure = orchestrate.ExitEncodeFailure
	// exitActionFailure remains for older tests; prefer ProcessExitFor.
	exitActionFailure = orchestrate.ExitInvalidRequest
)

func runCLI(arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		printCLIUsage(stderr)
		return exitUsage
	}
	switch arguments[0] {
	case "environment":
		return environmentCmd(arguments[1:], stdin, stdout, stderr, executeEnvironment)
	case "action":
		return actionCmd(arguments[1:], stdin, stdout, stderr, executeEvaluation)
	case "lifecycle":
		return lifecycleCmd(arguments[1:], stdin, stdout, stderr, executeServiceLifecycle)
	case "deploy":
		return deployCmd(arguments[1:], stdout, stderr)
	case "serve":
		serveCmd(arguments[1:])
		return exitOK
	case "token":
		tokenCmd(arguments[1:])
		return exitOK
	case "vm":
		vmCmd(arguments[1:])
		return exitOK
	case "help", "-h", "--help":
		printCLIUsage(stderr)
		return exitOK
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n", arguments[0])
		printCLIUsage(stderr)
		return exitUsage
	}
}

func printCLIUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "Usage: nvms <command> [flags]")
	fmt.Fprintln(stderr, "Commands:")
	fmt.Fprintln(stderr, "  deploy                 Deploy a workload to the specified tier")
	fmt.Fprintln(stderr, "  serve                  Start the NVMS daemon (HTTP over UDS)")
	fmt.Fprintln(stderr, "  token                  Manage bearer tokens (mint/list)")
	fmt.Fprintln(stderr, "  vm                     Manage VMs/sandboxes")
	fmt.Fprintln(stderr, "  action --request -     Run one evaluation action from stdin JSON")
	fmt.Fprintln(stderr, "  lifecycle --request -  Run one service lifecycle plan from stdin JSON")
	fmt.Fprintln(stderr, "  environment <op> --request -")
	fmt.Fprintln(stderr, "                         Plan/apply/verify host environment (op: plan|apply|verify)")
	fmt.Fprintln(stderr, "Exit codes: 0 ok, 2 usage, 3 invalid JSON, 4 invalid request,")
	fmt.Fprintln(stderr, "  5 encode failure, 6 contention, 7 host probe, 8 action runtime, 9 evidence/cleanup")
	fmt.Fprintln(stderr, "See pkg/orchestrate/EVALUATION.md for evaluation error codes.")
}

func deployCmd(arguments []string, stdout, stderr io.Writer) int {
	deploySet := flag.NewFlagSet("deploy", flag.ContinueOnError)
	deploySet.SetOutput(stderr)
	tier := deploySet.Int("tier", 1, "Deployment tier (1=WASM, 2=gVisor, 3=Firecracker)")
	config := deploySet.String("config", "nvms.yaml", "Path to deployment config")
	if err := deploySet.Parse(arguments); err != nil {
		return exitUsage
	}

	ctx := context.Background()
	if err := deploy.Deploy(ctx, *tier, *config); err != nil {
		fmt.Fprintf(stderr, "nvms deploy: %v\n", err)
		return orchestrate.ExitEvidence
	}
	fmt.Fprintf(stdout, "Deployment completed successfully (tier=%d, config=%s)\n", *tier, *config)
	return exitOK
}

type evaluationExecutor func(context.Context, orchestrate.EvaluationRequest) (orchestrate.EvaluationResult, error)

func actionCmd(arguments []string, stdin io.Reader, stdout, stderr io.Writer, execute evaluationExecutor) int {
	flags := flag.NewFlagSet("action", flag.ContinueOnError)
	flags.SetOutput(stderr)
	requestSource := flags.String("request", "", "JSON request source; only '-' (stdin) is supported")
	if err := flags.Parse(arguments); err != nil || *requestSource != "-" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "nvms action: usage: nvms action --request -")
		fmt.Fprintln(stderr, "nvms action: read one EvaluationRequest JSON from stdin; see pkg/orchestrate/EVALUATION.md")
		return exitUsage
	}
	decoder := json.NewDecoder(stdin)
	decoder.DisallowUnknownFields()
	var request orchestrate.EvaluationRequest
	if err := decoder.Decode(&request); err != nil {
		fmt.Fprintf(stderr, "nvms action: invalid_json: %v\n", err)
		return exitInvalidJSON
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		fmt.Fprintln(stderr, "nvms action: invalid_json: exactly one JSON request is required")
		return exitInvalidJSON
	}
	result, actionErr := execute(context.Background(), request)
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintf(stderr, "nvms action: encode_failed: %v\n", err)
		return exitEncodeFailure
	}
	if actionErr != nil {
		var evaluationErr *orchestrate.EvaluationError
		code := orchestrate.CodeEvaluationFailed
		message := actionErr
		if errors.As(actionErr, &evaluationErr) {
			code = evaluationErr.Code
			message = evaluationErr.Err
		}
		fmt.Fprintf(stderr, "nvms action: %s: %v\n", code, message)
		return orchestrate.ProcessExitFor(code)
	}
	return exitOK
}

type lifecycleExecutor func(context.Context, orchestrate.ServiceLifecycleRequest) (orchestrate.ServiceLifecycleResult, error)

func lifecycleCmd(arguments []string, stdin io.Reader, stdout, stderr io.Writer, execute lifecycleExecutor) int {
	flags := flag.NewFlagSet("lifecycle", flag.ContinueOnError)
	flags.SetOutput(stderr)
	requestSource := flags.String("request", "", "JSON request source; only '-' (stdin) is supported")
	if err := flags.Parse(arguments); err != nil || *requestSource != "-" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "nvms lifecycle: usage: nvms lifecycle --request -")
		return exitUsage
	}
	decoder := json.NewDecoder(stdin)
	decoder.DisallowUnknownFields()
	var request orchestrate.ServiceLifecycleRequest
	if err := decoder.Decode(&request); err != nil {
		fmt.Fprintf(stderr, "nvms lifecycle: invalid_json: %v\n", err)
		return exitInvalidJSON
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		fmt.Fprintln(stderr, "nvms lifecycle: invalid_json: exactly one JSON request is required")
		return exitInvalidJSON
	}
	result, lifecycleErr := execute(context.Background(), request)
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintf(stderr, "nvms lifecycle: encode_failed: %v\n", err)
		return exitEncodeFailure
	}
	if lifecycleErr != nil {
		var evaluationErr *orchestrate.EvaluationError
		code := orchestrate.CodeEvaluationFailed
		message := lifecycleErr
		if errors.As(lifecycleErr, &evaluationErr) {
			code = evaluationErr.Code
			message = evaluationErr.Err
		}
		fmt.Fprintf(stderr, "nvms lifecycle: %s: %v\n", code, message)
		return orchestrate.ProcessExitFor(code)
	}
	return exitOK
}

func executeServiceLifecycle(ctx context.Context, request orchestrate.ServiceLifecycleRequest) (orchestrate.ServiceLifecycleResult, error) {
	runner := gpu.ExecRunner{Timeout: 5 * time.Minute, MaxOutput: 16 << 20}
	action := orchestrate.ServiceLifecycleAction{Runner: runner}
	return action.Execute(ctx, request)
}

func executeEvaluation(ctx context.Context, request orchestrate.EvaluationRequest) (orchestrate.EvaluationResult, error) {
	timeout := time.Duration(request.TimeoutMillis) * time.Millisecond
	ttl := timeout + orchestrate.EvaluationReservationSkew
	if ttl > orchestrate.MaxEvaluationTimeout {
		ttl = orchestrate.MaxEvaluationTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ttl)
		defer cancel()
	}
	runner := gpu.ExecRunner{Timeout: timeout, MaxOutput: request.MaxOutputBytes}
	action := orchestrate.EvaluationAction{
		Registry:       nvmsruntime.NewBackendRegistry(),
		Inventory:      evaluationInventoryProvider(request, runner),
		Inspector:      orchestrate.HostEvaluationInspector{Runner: runner},
		Runner:         runner,
		Reservations:   &gpu.ReservationStore{Path: request.ReservationPath},
		ReservationTTL: ttl,
	}
	return action.Execute(ctx, request)
}

func evaluationInventoryProvider(request orchestrate.EvaluationRequest, runner gpu.CommandRunner) gpu.ReconciledInventoryProvider {
	adapters := []gpu.InventoryAdapter{gpu.WindowsInventoryAdapter{Runner: runner}}
	if request.WSLDistribution != "" {
		adapters = append(adapters, gpu.WSLInventoryAdapter{
			Runner: runner, Distribution: request.WSLDistribution,
		})
	}
	return gpu.ReconciledInventoryProvider{Adapters: adapters}
}

type environmentExecutor func(context.Context, string, environment.Request) (environment.Result, error)

func environmentCmd(arguments []string, stdin io.Reader, stdout, stderr io.Writer, execute environmentExecutor) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "nvms environment: usage: nvms environment <plan|apply|verify> --request -")
		return exitUsage
	}
	operation := arguments[0]
	flags := flag.NewFlagSet("environment", flag.ContinueOnError)
	flags.SetOutput(stderr)
	requestSource := flags.String("request", "", "JSON request source; only '-' (stdin) is supported")
	if err := flags.Parse(arguments[1:]); err != nil || *requestSource != "-" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "nvms environment: usage: nvms environment <plan|apply|verify> --request -")
		return exitUsage
	}
	switch operation {
	case "plan", "apply", "verify":
	default:
		fmt.Fprintf(stderr, "nvms environment: unknown operation %q\n", operation)
		return exitUsage
	}
	decoder := json.NewDecoder(stdin)
	decoder.DisallowUnknownFields()
	var request environment.Request
	if err := decoder.Decode(&request); err != nil {
		fmt.Fprintf(stderr, "nvms environment: invalid_json: %v\n", err)
		return exitInvalidJSON
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		fmt.Fprintln(stderr, "nvms environment: invalid_json: exactly one JSON request is required")
		return exitInvalidJSON
	}
	result, envErr := execute(context.Background(), operation, request)
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintf(stderr, "nvms environment: encode_failed: %v\n", err)
		return exitEncodeFailure
	}
	if envErr != nil {
		var providerErr *environment.ProviderError
		code := environment.CodeEnvironmentFailed
		message := envErr
		if errors.As(envErr, &providerErr) {
			code = providerErr.Code
			message = providerErr.Err
		}
		fmt.Fprintf(stderr, "nvms environment: %s: %v\n", code, message)
		return environment.ProcessExitFor(code)
	}
	return exitOK
}

func executeEnvironment(ctx context.Context, operation string, request environment.Request) (environment.Result, error) {
	timeout := 30 * time.Second
	runner := gpu.ExecRunner{Timeout: timeout, MaxOutput: 16 << 20}
	state := &environment.MemoryStateStore{}
	provider := environment.DefaultProvider(
		evaluationInventoryProvider(orchestrate.EvaluationRequest{WSLDistribution: request.WSLDistribution}, runner),
		runner,
		state,
	)
	switch operation {
	case "plan":
		return provider.Plan(ctx, request)
	case "apply":
		return provider.Apply(ctx, request)
	case "verify":
		return provider.Verify(ctx, request)
	default:
		return environment.Result{}, fmt.Errorf("unknown environment operation %q", operation)
	}
}

// serveCmd starts the provider-neutral NanoVMS HTTP control plane. Provider
// selection is explicit: the default remains tier-backed; provider-native
// Podman, Apple Containers, and WSL Containers are opt-in.
func serveCmd(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	socketPath := flags.String("socket", "", "UDS socket path")
	tokenFile := flags.String("token-file", "", "Path to token file")
	runBase := flags.String("run-base", "", "Runtime base directory")
	tierLevel := flags.Int("tier", 3, "Sandbox tier")
	provider := flags.String("provider", "tier", "Sandbox provider (tier, podman, apple-containers, or wsl-containers)")
	listenAddr := flags.String("listen", "", "TCP listen address")
	tlsCert := flags.String("tls-cert", "", "TLS certificate PEM")
	tlsKey := flags.String("tls-key", "", "TLS private key PEM")
	_ = flags.Parse(args)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *socketPath == "" && *listenAddr == "" {
		runDir := os.Getenv("XDG_RUNTIME_DIR")
		if runDir == "" {
			runDir = "/tmp"
		}
		*socketPath = runDir + "/nanovms/routed.sock"
	}
	if *tokenFile == "" {
		cfg := os.Getenv("XDG_CONFIG_DIR")
		if cfg == "" {
			cfg = "/etc/nanovms"
		}
		*tokenFile = cfg + "/tokens"
	}
	if *runBase == "" {
		*runBase = os.Getenv("XDG_RUNTIME_DIR")
		if *runBase == "" {
			*runBase = "/tmp"
		}
	}
	tm, err := token.NewManager(*tokenFile)
	if err != nil {
		log.Fatalf("token manager: %v", err)
	}
	var ln *listen.Listener
	if *listenAddr != "" {
		var tlsCfg *tls.Config
		if *tlsCert != "" || *tlsKey != "" {
			if *tlsCert == "" || *tlsKey == "" {
				log.Fatal("both --tls-cert and --tls-key are required")
			}
			cert, loadErr := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
			if loadErr != nil {
				log.Fatalf("load TLS keypair: %v", loadErr)
			}
			tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		}
		ln, err = listen.NewTCP(ctx, *listenAddr, tlsCfg)
	} else {
		ln, err = listen.NewUDS(ctx, *socketPath, *runBase)
	}
	if err != nil {
		log.Fatalf("listener: %v", err)
	}
	defer ln.Close()
	adapter, err := adapters.NewProvider(strings.ToLower(strings.TrimSpace(*provider)), *tierLevel)
	if err != nil {
		log.Fatalf("provider %s: %v", *provider, err)
	}
	auditDir := os.TempDir()
	if p := os.Getenv("NVMS_AUDIT_PATH"); p != "" {
		auditDir = p
	}
	audit := api.NewAuditLogger(auditDir)
	if err := api.Serve(ctx, ln, api.Handlers{Port: adapter, Token: tm, AuditLog: audit}); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func tokenCmd(args []string) {
	flags := flag.NewFlagSet("token", flag.ExitOnError)
	_ = flags.Parse(args)
	if flags.NArg() == 0 || flags.Arg(0) == "help" {
		fmt.Println("Usage: nvms token mint|list")
		return
	}
	switch flags.Arg(0) {
	case "mint":
		tok, err := token.MintToken()
		if err != nil {
			log.Fatalf("mint: %v", err)
		}
		fmt.Println(tok)
	case "list":
		fmt.Println("token list: inspect the configured token file")
	default:
		log.Fatalf("nvms token: unknown subcommand %q", flags.Arg(0))
	}
}

func vmCmd(args []string) {
	flags := flag.NewFlagSet("vm", flag.ExitOnError)
	socket := flags.String("socket", "", "UDS socket path")
	_ = flags.Parse(args)
	if flags.NArg() == 0 || flags.Arg(0) == "help" {
		fmt.Println("Usage: nvms vm --socket <path> list|exec|logs")
		return
	}
	client := sandbox.NewClient(*socket)
	switch flags.Arg(0) {
	case "list":
		items, err := client.ListSandboxes(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			fmt.Printf("%s\t%s\t%s\n", item.ID, item.Name, item.Status)
		}
	case "exec":
		if flags.NArg() < 3 {
			log.Fatal("usage: nvms vm exec <id> <cmd...>")
		}
		out, err := client.Exec(context.Background(), flags.Arg(1), flags.Args()[2:])
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
		_, _ = io.Copy(os.Stdout, out)
	case "logs":
		if flags.NArg() < 2 {
			log.Fatal("usage: nvms vm logs <id>")
		}
		out, err := client.Logs(context.Background(), flags.Arg(1), false)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
		_, _ = io.Copy(os.Stdout, out)
	default:
		log.Fatalf("nvms vm: unknown subcommand %q", flags.Arg(0))
	}
}
