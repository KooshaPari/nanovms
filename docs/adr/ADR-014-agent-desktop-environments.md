# ADR-014: Agent Desktop Environments & Computer Use

## Status

Proposed

## Context

NanoVMS needs to support AI agents that perform computer use:
- Desktop environments (browser, file management)
- Virtual displays (headless X11/Wayland)
- Screen capture and input simulation
- Multi-agent orchestration

## Decision

### Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                     Agent Desktop Architecture                            │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    AI Agent Layer                               │  │
│  │                                                                       │  │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │  │
│  │   │  Browser     │  │  File       │  │  Terminal   │   │  │
│  │   │  Agent       │  │  Manager    │  │  Agent     │   │  │
│  │   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │  │
│  │          │                   │                   │             │  │
│  │          └───────────────────┼───────────────────┘             │  │
│  │                              ▼                                 │  │
│  │                    ┌──────────────┐                         │  │
│  │                    │  Computer   │                         │  │
│  │                    │  Use API    │                         │  │
│  │                    └──────┬───────┘                         │  │
│  └───────────────────────────┼─────────────────────────────────┘  │
│                              │                                       │
│  ┌───────────────────────────┼─────────────────────────────────┐  │
│  │                    Desktop VM                                   │  │
│  │                              ▼                                 │  │
│  │   ┌──────────────────────────────────────────────────────┐  │  │
│  │   │                    X11/Wayland Server                     │  │  │
│  │   │                                                               │  │  │
│  │   │   ┌────────────┐  ┌────────────┐  ┌────────────┐  │  │  │
│  │   │   │  Firefox  │  │  Thunar   │  │  Terminal  │  │  │  │
│  │   │   │  Browser  │  │  File Mgr  │  │  xterm    │  │  │  │
│  │   │   └────────────┘  └────────────┘  └────────────┘  │  │  │
│  │   │                                                               │  │  │
│  │   │   ┌──────────────────────────────────────────────┐  │  │  │
│  │   │   │           Screen Capture (VNC/RDP)                  │  │  │  │
│  │   │   └──────────────────────────────────────────────┘  │  │  │
│  │   └──────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                              │                                       │
│  ┌───────────────────────────┼─────────────────────────────────┐  │
│  │                    NanoVMS Core                               │  │
│  │                                                                       │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │   │  VM Manager │  │  Display   │  │  Input     │   │  │
│  │   │  (Firecracker)│  │  Bridge   │  │  Bridge   │   │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Agent Desktop VM

```go
// pkg/agent/desktop/desktop.go
package desktop

import (
    "context"
    "fmt"
    "image"
    "time"

    "github.com/KooshaPari/nanovms/pkg/vm"
    "github.com/KooshaPari/nanovms/pkg/display"
    "github.com/KooshaPari/nanovms/pkg/input"
)

// DesktopVM is a VM configured for AI agent use
type DesktopVM struct {
    *vm.Instance
    display  *display.VNCBridge
    input   *input.X11Bridge
    desktop DesktopConfig
}

// DesktopConfig defines the desktop environment
type DesktopConfig struct {
    // Display settings
    Resolution image.Point  // e.g., 1920x1080
    DPI       int          // e.g., 96

    // Desktop environment
    DesktopEnv string      // "xfce", "gnome", "kde", "lxde"

    // Browser settings
    Browser BrowserConfig

    // Input settings
    InputMode string       // "x11", "wayland", "vnc"

    // Resources
    CPU       int
    MemoryGB  int
    DiskGB    int
}

type BrowserConfig struct {
    Browser   string       // "firefox", "chromium", "webkit"
    UserAgent string
    Headless  bool
    Proxy     string
}

// LaunchDesktop creates and starts a desktop VM for agents
func LaunchDesktop(ctx context.Context, cfg DesktopConfig) (*DesktopVM, error) {
    // Create VM with desktop resources
    vmInst, err := vm.Create(ctx, &vm.Config{
        Name:     fmt.Sprintf("agent-desktop-%d", time.Now().Unix()),
        Flavor:   "microvm",
        CPU:      cfg.CPU,
        MemoryGB: cfg.MemoryGB,
        DiskGB:   cfg.DiskGB,
    })
    if err != nil {
        return nil, fmt.Errorf("create vm: %w", err)
    }

    // Start VM
    if err := vmInst.Start(ctx); err != nil {
        return nil, fmt.Errorf("start vm: %w", err)
    }

    // Setup display bridge
    displayBridge, err := display.NewVNCBridge(vmInst, display.VNCConfig{
        Port:     5900,
        Password: "",  // No password for agent access
        Clipboard: true,
    })
    if err != nil {
        return nil, fmt.Errorf("setup display: %w", err)
    }

    // Setup input bridge
    inputBridge, err := input.NewX11Bridge(vmInst, input.X11Config{
        Display: ":0",
    })
    if err != nil {
        return nil, fmt.Errorf("setup input: %w", err)
    }

    return &DesktopVM{
        Instance: vmInst,
        display:  displayBridge,
        input:   inputBridge,
        desktop:  cfg,
    }, nil
}

// CaptureScreen captures the current screen state
func (d *DesktopVM) CaptureScreen(ctx context.Context) (*image.RGBA, error) {
    return d.display.Capture(ctx)
}

// Click performs a mouse click at the specified coordinates
func (d *DesktopVM) Click(ctx context.Context, x, y int, button input.MouseButton) error {
    return d.input.Click(ctx, x, y, button)
}

// Type simulates keyboard input
func (d *DesktopVM) Type(ctx context.Context, text string) error {
    return d.input.Type(ctx, text)
}

// PressKey presses a keyboard key
func (d *DesktopVM) PressKey(ctx context.Context, key input.Key) error {
    return d.input.PressKey(ctx, key)
}
```

### Computer Use API

```go
// pkg/agent/computer/computer.go
package computer

import (
    "context"
    "image"
    "time"

    "github.com/KooshaPari/nanovms/pkg/agent/desktop"
)

// ComputerUse implements the computer use API for AI agents
type ComputerUse struct {
    vmPool  *DesktopVMPool
    browser *BrowserController
}

// DesktopVMPool manages a pool of desktop VMs for agents
type DesktopVMPool struct {
    available chan *desktop.DesktopVM
    active    map[string]*desktop.DesktopVM
    maxVMs    int
}

// Acquire gets an available desktop VM
func (p *DesktopVMPool) Acquire(ctx context.Context) (*desktop.DesktopVM, error) {
    select {
    case vm := <-p.available:
        p.active[vm.ID()] = vm
        return vm, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Create new VM if under limit
        if len(p.active) < p.maxVMs {
            vm, err := desktop.LaunchDesktop(ctx, desktop.DesktopConfig{
                Resolution: image.Point{1920, 1080},
                DPI:        96,
                DesktopEnv: "xfce",
                CPU:        4,
                MemoryGB:   8,
                DiskGB:     50,
            })
            if err != nil {
                return nil, err
            }
            p.active[vm.ID()] = vm
            return vm, nil
        }
        // Wait for available VM
        return p.Acquire(ctx)  // Recursive wait
    }
}

// Release returns a VM to the pool
func (p *DesktopVMPool) Release(vm *desktop.DesktopVM) {
    delete(p.active, vm.ID())
    p.available <- vm  // Return to available pool
}

// Screenshot captures the current screen
func (c *ComputerUse) Screenshot(ctx context.Context) (*image.RGBA, error) {
    vm, err := c.vmPool.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    defer c.vmPool.Release(vm)

    return vm.CaptureScreen(ctx)
}

// Click clicks at the specified coordinates
func (c *ComputerUse) Click(ctx context.Context, x, y int) error {
    vm, err := c.vmPool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer c.vmPool.Release(vm)

    return vm.Click(ctx, x, y, input.LeftButton)
}

// Type sends text input
func (c *ComputerUse) Type(ctx context.Context, text string) error {
    vm, err := c.vmPool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer c.vmPool.Release(vm)

    return vm.Type(ctx, text)
}

// OpenURL opens a URL in the browser
func (c *ComputerUse) OpenURL(ctx context.Context, url string) error {
    vm, err := c.vmPool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer c.vmPool.Release(vm)

    return vm.Exec(ctx, fmt.Sprintf("firefox '%s'", url))
}
```

### Browser Agent

```go
// pkg/agent/browser/browser.go
package browser

import (
    "context"
    "fmt"
    "regexp"
    "strings"

    "github.com/KooshaPari/nanovms/pkg/vm"
)

type BrowserAgent struct {
    vm *vm.Instance
}

// Navigation represents a browser navigation
type Navigation struct {
    URL     string
    Title   string
    LoadedAt time.Time
}

// GetCurrentURL gets the current browser URL
func (b *BrowserAgent) GetCurrentURL(ctx context.Context) (string, error) {
    output, err := b.vm.Exec(ctx, "xdotool getactivewindow getwindowname")
    if err != nil {
        return "", err
    }
    // Parse URL from window title (format: "Title - URL")
    parts := strings.Split(strings.TrimSpace(output), " - ")
    if len(parts) > 1 {
        return parts[len(parts)-1], nil
    }
    return "", nil
}

// ClickElement clicks an element by selector
func (b *BrowserAgent) ClickElement(ctx context.Context, selector string) error {
    // Use xdotool to click at position
    cmd := fmt.Sprintf(`xdotool click --window %s $(xdotool getactivewindow getname | grep -o '%s' | head -1)`, selector, selector)
    _, err := b.vm.Exec(ctx, cmd)
    return err
}

// TypeInField types text into an input field
func (b *BrowserAgent) TypeInField(ctx context.Context, selector string, text string) error {
    // Focus field and type
    cmd := fmt.Sprintf(`xdotool search --name '%s' windowfocus type '%s'`, selector, text)
    _, err := b.vm.Exec(ctx, cmd)
    return err
}

// WaitForSelector waits for an element to appear
func (b *BrowserAgent) WaitForSelector(ctx context.Context, selector string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        output, err := b.vm.Exec(ctx, fmt.Sprintf(`xdotool getactivewindow getwindowname | grep -c '%s'`, selector))
        if err == nil && strings.TrimSpace(output) == "1" {
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }

    return fmt.Errorf("timeout waiting for selector: %s", selector)
}

// ExtractData extracts structured data from page
func (b *BrowserAgent) ExtractData(ctx context.Context, pattern string) (string, error) {
    // Use xclip to get page content
    output, err := b.vm.Exec(ctx, "xclip -selection clipboard -o")
    if err != nil {
        return "", err
    }

    re := regexp.MustCompile(pattern)
    match := re.FindString(output)
    return match, nil
}
```

### Multi-Agent Orchestration

```go
// pkg/agent/orchestrator/orchestrator.go
package orchestrator

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/KooshaPari/nanovms/pkg/agent/desktop"
)

type Agent struct {
    ID       string
    Name     string
    Desktop  *desktop.DesktopVM
    TaskCh   chan *Task
    ResultCh chan *Result
    state    AgentState
}

type Task struct {
    ID          string
    Type        string        // "browse", "file", "terminal", "custom"
    Instructions string
    Context     map[string]interface{}
    Deadline    time.Time
}

type Result struct {
    TaskID  string
    Success bool
    Data    interface{}
    Error   error
}

type AgentState int

const (
    AgentIdle AgentState = iota
    AgentWorking
    AgentWaiting
    AgentError
)

// Orchestrator manages multiple agent desktops
type Orchestrator struct {
    agents    map[string]*Agent
    desktopPool *desktop.DesktopVMPool
    mutex     sync.RWMutex
}

// CreateAgent creates a new agent with desktop VM
func (o *Orchestrator) CreateAgent(ctx context.Context, name string, cfg desktop.DesktopConfig) (*Agent, error) {
    o.mutex.Lock()
    defer o.mutex.Unlock()

    vm, err := o.desktopPool.Acquire(ctx)
    if err != nil {
        return nil, err
    }

    agent := &Agent{
        ID:       fmt.Sprintf("agent-%d", time.Now().UnixNano()),
        Name:     name,
        Desktop:  vm,
        TaskCh:   make(chan *Task, 10),
        ResultCh: make(chan *Result, 10),
        state:    AgentIdle,
    }

    o.agents[agent.ID] = agent

    // Start agent worker
    go agent.run()

    return agent, nil
}

// SubmitTask submits a task to an agent
func (o *Orchestrator) SubmitTask(ctx context.Context, agentID string, task *Task) error {
    o.mutex.RLock()
    agent, ok := o.agents[agentID]
    o.mutex.RUnlock()

    if !ok {
        return fmt.Errorf("agent not found: %s", agentID)
    }

    select {
    case agent.TaskCh <- task:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// BroadcastTask sends task to all agents
func (o *Orchestrator) BroadcastTask(ctx context.Context, task *Task) {
    o.mutex.RLock()
    defer o.mutex.RUnlock()

    for _, agent := range o.agents {
        select {
        case agent.TaskCh <- task:
        default:
            // Skip if agent queue full
        }
    }
}

// Agent worker loop
func (a *Agent) run() {
    for task := range a.TaskCh {
        a.state = AgentWorking

        result := &Result{
            TaskID:  task.ID,
            Success: true,
        }

        switch task.Type {
        case "browse":
            result.Data, result.Error = a.executeBrowse(task)
        case "file":
            result.Data, result.Error = a.executeFile(task)
        case "terminal":
            result.Data, result.Error = a.executeTerminal(task)
        case "screenshot":
            img, err := a.Desktop.CaptureScreen(context.Background())
            result.Data = img
            result.Error = err
        default:
            result.Success = false
            result.Error = fmt.Errorf("unknown task type: %s", task.Type)
        }

        if result.Error != nil {
            a.state = AgentError
        } else {
            a.state = AgentIdle
        }

        a.ResultCh <- result
    }
}

func (a *Agent) executeBrowse(task *Task) (interface{}, error) {
    ctx := context.WithTimeout(context.Background(), 30*time.Second)

    // Extract URL from instructions
    url := task.Instructions  // Simplified

    // Open URL
    if err := a.Desktop.Exec(ctx, fmt.Sprintf("firefox '%s'", url)); err != nil {
        return nil, err
    }

    // Wait for page load
    time.Sleep(2 * time.Second)

    // Take screenshot
    img, err := a.Desktop.CaptureScreen(ctx)
    if err != nil {
        return nil, err
    }

    return img, nil
}
```

### VNC Bridge

```go
// pkg/display/vnc/bridge.go
package vnc

import (
    "context"
    "fmt"
    "image"
    "image/png"
    "net"
    "sync"
    "time"

    "github.com/KooshaPari/nanovms/pkg/vm"
)

// VNCBridge provides VNC access to desktop VM
type VNCBridge struct {
    vm     *vm.Instance
    port   int
    conn   net.Conn
    mutex  sync.Mutex
}

// NewVNCBridge creates a VNC bridge to the VM
func NewVNCBridge(vm *vm.Instance, port int) (*VNCBridge, error) {
    // Connect via VNC port forwarding
    vncPort := vm.ForwardPort(port)
    if vncPort == 0 {
        return nil, fmt.Errorf("failed to forward VNC port")
    }

    conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", vncPort), 5*time.Second)
    if err != nil {
        return nil, fmt.Errorf("vnc connect: %w", err)
    }

    return &VNCBridge{
        vm:   vm,
        port: port,
        conn: conn,
    }, nil
}

// Capture captures the current screen via VNC
func (v *VNCBridge) Capture(ctx context.Context) (*image.RGBA, error) {
    v.mutex.Lock()
    defer v.mutex.Unlock()

    // Send framebuffer request
    req := []byte{0, 0, 0, 0, 0, 0}  // FramebufferRequest
    if _, err := v.conn.Write(req); err != nil {
        return nil, err
    }

    // Read framebuffer update
    buf := make([]byte, 1024*1024)  // 1MB buffer
    n, err := v.conn.Read(buf)
    if err != nil {
        return nil, err
    }

    // Decode VNC data to image (simplified)
    return v.decodeFramebuffer(buf[:n])
}

// MouseMove moves the mouse cursor
func (v *VNCBridge) MouseMove(x, y int) error {
    v.mutex.Lock()
    defer v.mutex.Unlock()

    // VNC pointer event: button mask (1 byte) + x (2 bytes) + y (2 bytes)
    msg := []byte{0, byte(x & 0xFF), byte((x >> 8) & 0xFF), byte(y & 0xFF), byte((y >> 8) & 0xFF)}
    _, err := v.conn.Write(msg)
    return err
}

func (v *VNCBridge) decodeFramebuffer(data []byte) (*image.RGBA, error) {
    // Simplified: convert raw pixel data to RGBA
    // Real implementation would parse VNC encoding types
    img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))

    offset := 0
    for y := 0; y < 1080; y++ {
        for x := 0; x < 1920; x++ {
            if offset+3 < len(data) {
                img.Set(x, y, image.RGBA{
                    R: data[offset],
                    G: data[offset+1],
                    B: data[offset+2],
                    A: 255,
                })
                offset += 4
            }
        }
    }

    return img, nil
}
```

### Input Bridge

```go
// pkg/input/x11/bridge.go
package x11

import (
    "context"
    "fmt"

    "github.com/KooshaPari/nanovms/pkg/vm"
)

type Key uint

const (
    KeyEnter Key = 36
    KeyEscape Key = 9
    KeyTab Key = 23
    // ... more keys
)

type MouseButton int

const (
    LeftButton MouseButton = 1
    MiddleButton MouseButton = 2
    RightButton MouseButton = 3
)

// X11Bridge provides X11 input to VM
type X11Bridge struct {
    vm      *vm.Instance
    display string
}

// NewX11Bridge creates X11 input bridge
func NewX11Bridge(vm *vm.Instance, display string) (*X11Bridge, error) {
    return &X11Bridge{
        vm:      vm,
        display: display,
    }, nil
}

// Click performs mouse click
func (x *X11Bridge) Click(ctx context.Context, xPos, yPos int, button MouseButton) error {
    cmd := fmt.Sprintf("xdotool mousemove %d %d click %d", xPos, yPos, button)
    _, err := x.vm.Exec(ctx, cmd)
    return err
}

// Type sends text input
func (x *X11Bridge) Type(ctx context.Context, text string) error {
    cmd := fmt.Sprintf("xdotool type --delay 50 '%s'", text)
    _, err := x.vm.Exec(ctx, cmd)
    return err
}

// PressKey presses a single key
func (x *X11Bridge) PressKey(ctx context.Context, key Key) error {
    cmd := fmt.Sprintf("xdotool key %d", key)
    _, err := x.vm.Exec(ctx, cmd)
    return err
}

// Scroll performs mouse scroll
func (x *X11Bridge) Scroll(ctx context.Context, xPos, yPos int, deltaY int) error {
    cmd := fmt.Sprintf("xdotool mousemove %d %d click --repeat 2 button4", xPos, yPos)
    if deltaY < 0 {
        cmd = fmt.Sprintf("xdotool mousemove %d %d click --repeat %d button5", xPos, yPos, -deltaY)
    }
    _, err := x.vm.Exec(ctx, cmd)
    return err
}
```

### Performance Targets

| Metric | Target | Method |
|--------|--------|--------|
| **Screen capture** | <100ms | VNC framebuffer |
| **Input latency** | <50ms | X11 socket |
| **VM start** | <5s | Pre-warmed pool |
| **Agent switch** | <500ms | Pool acquire |
| **Concurrent agents** | 8 per host | VM pool quota |
| **Memory per agent** | 4-8GB | Desktop VM |

### Consequences

### Positive
- Standardized computer use API for AI agents
- Isolated agent environments (one VM per agent)
- VNC/X11 provides universal display access
- Pre-warmed pool reduces latency

### Negative
- High resource usage per agent (4-8GB RAM)
- VNC latency may affect pixel-accurate tasks
- Browser automation limited to installed browsers
- No GPU acceleration (headless mode)

## References

- Anthropic Computer Use API
- Browserbase / Skyvern
- OpenAI Computer Use
- AutoGPT Browser Use
