package mac

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/kooshapari/nanovms/internal/domain"
)

// List lists all VMs.
func (a *Adapter) List(ctx context.Context) ([]domain.Sandbox, error) {
	switch a.Name() {
	case "firecracker":
		return a.listFirecrackerVMs(ctx)
	case "hyperkit":
		return a.listHyperKitVMs(ctx)
	default: // lima/colima
		return a.listLimaVMs(ctx)
	}
}

// listLimaVMs lists Lima/Colima VMs.
func (a *Adapter) listLimaVMs(ctx context.Context) ([]domain.Sandbox, error) {
	cmd := exec.CommandContext(ctx, a.limaPath, "list", "--json")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var vms []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &vms); err != nil {
		return nil, err
	}

	result := make([]domain.Sandbox, 0, len(vms))
	for _, vm := range vms {
		result = append(result, domain.Sandbox{
			ID:     vm.Name,
			Name:   vm.Name,
			Status: domain.ParseStatus(vm.Status),
		})
	}
	return result, nil
}

// listHyperKitVMs lists HyperKit VMs.
func (a *Adapter) listHyperKitVMs(ctx context.Context) ([]domain.Sandbox, error) {
	cmd := exec.CommandContext(ctx, a.hyperkitPath, "list")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Parse hyperkit list output
	var result []domain.Sandbox
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VM:") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				result = append(result, domain.Sandbox{
					ID:     parts[1],
					Name:   parts[1],
					Status: domain.StatusRunning,
				})
			}
		}
	}
	return result, nil
}

// listFirecrackerVMs lists Firecracker MicroVMs.
func (a *Adapter) listFirecrackerVMs(ctx context.Context) ([]domain.Sandbox, error) {
	// Firecracker VMs are listed via API socket files
	cmd := exec.CommandContext(ctx, "ls", "/var/run/firecracker/")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, nil // No VMs running
	}

	var result []domain.Sandbox
	sockets := strings.Split(out.String(), "\n")
	for _, sock := range sockets {
		if strings.HasSuffix(sock, ".sock") {
			name := strings.TrimSuffix(sock, ".sock")
			result = append(result, domain.Sandbox{
				ID:     name,
				Name:   name,
				Status: domain.StatusRunning, // Assume running if socket exists
			})
		}
	}
	return result, nil
}
