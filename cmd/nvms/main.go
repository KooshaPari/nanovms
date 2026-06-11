package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kooshapari/nanovms/pkg/deploy"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nvms <command> [flags]")
		fmt.Println("Commands:")
		fmt.Println("  deploy    Deploy a workload to the specified tier")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "deploy":
		deployCmd()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
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
