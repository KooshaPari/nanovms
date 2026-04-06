# ADR-013: Game Automation Testing Architecture

## Status

Proposed

## Context

NanoVMS needs to support parallel game automation testing:
- Headless Steam + tokens
- Pre-copied compressed images
- <10s startup target
- Unity ECS / BepInEx mod integration

## Decision

### Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                     Game Automation Testing Architecture                 │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Test Runner Controller                          │  │
│  │                                                                       │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │  │
│  │   │   pytest    │  │  Ginkgo    │  │  testify    │    │  │
│  │   │  Python    │  │   Go       │  │   Go       │    │  │
│  │   │  test suite │  │  BDD tests │  │  Go tests  │    │  │
│  │   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │  │
│  │          │                 │                 │               │  │
│  │          └─────────────────┼─────────────────┘               │  │
│  │                            ▼                                   │  │
│  │                   ┌────────────────┐                    │  │
│  │                   │  NanoVMS       │                    │  │
│  │                   │  Orchestrator   │                    │  │
│  │                   └────────┬────────┘                    │  │
│  │                            │                                 │  │
│  └────────────────────────────┼─────────────────────────────────┘  │
│                               │                                       │
│                               ▼                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Game VM Pool                                    │  │
│  │                                                                       │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │  │
│  │   │  Game VM 1  │  │  Game VM 2  │  │  Game VM N  │    │  │
│  │   │  Steam +    │  │  Steam +    │  │  Steam +    │    │  │
│  │   │  Tokens     │  │  Tokens     │  │  Tokens     │    │  │
│  │   │  BepInEx   │  │  BepInEx   │  │  BepInEx   │    │  │
│  │   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │  │
│  │          │                 │                 │               │  │
│  │          ▼                 ▼                 ▼               │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │  │
│  │   │  Game Proc  │  │  Game Proc  │  │  Game Proc  │    │  │
│  │   │  (Unity/    │  │  (Unity/    │  │  (Unity/    │    │  │
│  │   │   Unreal)   │  │   Unreal)   │  │   Unreal)   │    │  │
│  │   └─────────────┘  └─────────────┘  └─────────────┘    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                               │                                       │
│                               ▼                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Automation Layer                                │  │
│  │                                                                       │  │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │  │
│  │   │  Memory        │  │  API            │  │  Pixel          │ │  │
│  │   │  Scanner       │  │  Interceptor    │  │  Matcher       │ │  │
│  │   │  (CheatEngine) │  │  (Mitmproxy)   │  │  (OpenCV)      │ │  │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘ │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                               │                                       │
│                               ▼                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Results Collector                               │  │
│  │                                                                       │  │
│  │   JUnit XML ──► CI/CD          Allure ──► Reports             │  │
│  │   Temporal ──► Workflow       Prometheus ──► Metrics           │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Test Runner Integration

```go
// pkg/game/tester/tester.go
package tester

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/KooshaPari/nanovms/pkg/vm"
    "github.com/KooshaPari/nanovms/pkg/game"
)

// GameTester manages parallel game test execution
type GameTester struct {
    pool       *VMCluster
    runner     TestRunner
    results    *ResultCollector
    parallel   int
    mutex      sync.Mutex
}

// TestRunner interface for different test frameworks
type TestRunner interface {
    // Run executes test suite
    Run(ctx context.Context, tests []TestCase) (*TestResult, error)
    // ParseResults parses framework-specific output
    ParseResults(output []byte) ([]TestResult, error)
}

// Parallel execution
func (t *GameTester) RunParallel(ctx context.Context, testSuite string) (*TestReport, error) {
    tests, err := t.discoverTests(testSuite)
    if err != nil {
        return nil, err
    }

    // Create semaphore for parallel limit
    sem := make(chan struct{}, t.parallel)

    var wg sync.WaitGroup
    results := make(chan *TestResult, len(tests))
    errors := make(chan error, len(tests))

    for _, test := range tests {
        wg.Add(1)
        go func(test TestCase) {
            defer wg.Done()

            // Acquire VM slot
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
            case <-ctx.Done():
                errors <- ctx.Err()
                return
            }

            // Get pre-warmed VM
            guestVM, err := t.pool.Acquire(ctx, game.VMConfig{
                Flavor:   "microvm",
                PreCached: true,  // Use pre-copied image
            })
            if err != nil {
                errors <- err
                return
            }
            defer t.pool.Release(guestVM)

            // Run test in VM
            result, err := t.runTest(ctx, guestVM, test)
            if err != nil {
                errors <- err
                return
            }

            results <- result
        }(test)
    }

    wg.Wait()
    close(results)
    close(errors)

    return t.compileReport(results, errors), nil
}
```

### VM Pre-Warming & Snapshots

```go
// pkg/game/snapshot/snapshot.go
package snapshot

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/KooshaPari/nanovms/pkg/vm"
    "github.com/KooshaPari/nanovms/pkg/storage"
)

type SnapshotManager struct {
    baseImage  string
    snapshotDir string
    storage    *storage.Manager
}

// PreCopy prepares game image with all dependencies
func (sm *SnapshotManager) PreCopy(ctx context.Context, game GameConfig) error {
    baseDir := filepath.Join(sm.snapshotDir, game.ID)

    // Create working directory
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        return fmt.Errorf("create dir: %w", err)
    }

    // Create VM instance
    vmInst, err := sm.createFromBase(ctx, game, baseDir)
    if err != nil {
        return fmt.Errorf("create vm: %w", err)
    }
    defer vmInst.Cleanup()

    // Install game
    if err := sm.installGame(ctx, vmInst, game); err != nil {
        return fmt.Errorf("install game: %w", err)
    }

    // Install Steam + tokens
    if err := sm.installSteam(ctx, vmInst, game.SteamTokens); err != nil {
        return fmt.Errorf("install steam: %w", err)
    }

    // Install BepInEx mods
    if err := sm.installMods(ctx, vmInst, game.Mods); err != nil {
        return fmt.Errorf("install mods: %w", err)
    }

    // Pre-warm game
    if err := sm.prewarm(ctx, vmInst, game.WarmupScript); err != nil {
        return fmt.Errorf("prewarm: %w", err)
    }

    // Create snapshot
    snapshotPath := filepath.Join(baseDir, "snapshot.qcow2")
    if err := vmInst.CreateSnapshot(ctx, snapshotPath); err != nil {
        return fmt.Errorf("create snapshot: %w", err)
    }

    // Compress for distribution
    compressedPath := filepath.Join(sm.snapshotDir, fmt.Sprintf("%s.gz", game.ID))
    if err := sm.compress(ctx, snapshotPath, compressedPath); err != nil {
        return fmt.Errorf("compress: %w", err)
    }

    return nil
}

// RestoreVM restores VM from pre-copied snapshot
func (sm *SnapshotManager) RestoreVM(ctx context.Context, gameID string) (*vm.Instance, error) {
    snapshotPath := filepath.Join(sm.snapshotDir, fmt.Sprintf("%s.gz", gameID))
    workDir := filepath.Join(sm.snapshotDir, "temp", gameID)

    // Decompress if needed
    if err := sm.decompress(ctx, snapshotPath, workDir); err != nil {
        return nil, fmt.Errorf("decompress: %w", err)
    }

    // Create VM from snapshot
    vmInst, err := sm.createFromSnapshot(ctx, gameID, workDir)
    if err != nil {
        return nil, fmt.Errorf("create from snapshot: %w", err)
    }

    return vmInst, nil
}

// Target: <10s from cold start to game running
func (sm *SnapshotManager) BenchmarkStartup(gameID string) (time.Duration, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    start := time.Now()

    vmInst, err := sm.RestoreVM(ctx, gameID)
    if err != nil {
        return 0, err
    }
    defer vmInst.Cleanup()

    // Wait for game to be ready
    if err := vmInst.WaitForProcess(ctx, "game.exe", 10*time.Second); err != nil {
        return 0, err
    }

    return time.Since(start), nil
}
```

### Steam Integration

```go
// pkg/game/steam/steam.go
package steam

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "regexp"
    "time"

    "github.com/KooshaPari/nanovms/pkg/vm"
)

type SteamConfig struct {
    Username    string
    Password   string
    AuthCode   string   // 2FA code
    RefreshToken string
    GameID     uint64
}

type SteamManager struct {
    apiKey      string
    workshopDir string
}

// Login authenticates with Steam
func (sm *SteamManager) Login(ctx context.Context, cfg SteamConfig) (*Session, error) {
    // Check for cached refresh token
    if cfg.RefreshToken != "" {
        return sm.refreshLogin(ctx, cfg.RefreshToken)
    }

    // Full login flow
    return sm.passwordLogin(ctx, cfg)
}

// InstallGame installs game via SteamCMD
func (sm *SteamManager) InstallGame(ctx context.Context, vm *vm.Instance, appID uint64, depot string) error {
    // Mount SteamCMD directory
    if err := vm.Mount(ctx, "/steamcmd", sm.steamCMDPath); err != nil {
        return err
    }

    // Login anonymous for public games, or authenticated for private
    loginCmd := "steamcmd +login anonymous +force_install_dir /games/%d +app_update %d validate +quit"
    if vm.Authenticated() {
        loginCmd = fmt.Sprintf("steamcmd +login %s %s +force_install_dir /games/%d +app_update %d validate +quit",
            cfg.Username, cfg.Password, appID)
    }

    _, err := vm.Exec(ctx, loginCmd)
    return err
}

// StartHeadless starts Steam in headless mode with token
func (sm *SteamManager) StartHeadless(ctx context.Context, vm *vm.Instance, token string) error {
    // Set environment
    env := map[string]string{
        "STEAM_AUTH_TOKEN": token,
        "STEAM_API_TOKEN": sm.apiKey,
        "DISPLAY":         ":0",  // Virtual display
    }

    for k, v := range env {
        if err := vm.SetEnv(ctx, k, v); err != nil {
            return err
        }
    }

    // Start virtual display (Xvfb)
    if err := vm.ExecBackground(ctx, "Xvfb :0 -screen 0 1024x768x24"); err != nil {
        return fmt.Errorf("start Xvfb: %w", err)
    }

    // Start Steam in background
    cmd := fmt.Sprintf("steam -applaunch %d -no-browser -silent", sm.gameID)
    return vm.ExecBackground(ctx, cmd)
}

// TokenManager handles token lifecycle
type TokenManager struct {
    tokens     map[string]*Token
    refreshFn  func(token string) (*Token, error)
}

type Token struct {
    Value       string
    ExpiresAt   time.Time
    Permissions []string
}

func (tm *TokenManager) AcquireToken(ctx context.Context, userID string) (string, error) {
    tm.mutex.Lock()
    defer tm.mutex.Unlock()

    if token, ok := tm.tokens[userID]; ok && time.Until(token.ExpiresAt) > 5*time.Minute {
        return token.Value, nil
    }

    // Refresh token
    newToken, err := tm.refreshFn(tm.tokens[userID].Value)
    if err != nil {
        return "", err
    }

    tm.tokens[userID] = newToken
    return newToken.Value, nil
}
```

### BepInEx Integration

```go
// pkg/game/bepinex/bepinex.go
package bepinex

import (
    "context"
    "fmt"
    "path/filepath"

    "github.com/KooshaPari/nanovms/pkg/vm"
)

type ModConfig struct {
    Name        string
    Version     string
    DownloadURL string
    Files       []string  // Files to inject
    ConfigPatch map[string]string  // Config patches
}

type BepInExManager struct {
    baseDir string
}

// Install installs BepInEx into game directory
func (b *BepInExManager) Install(ctx context.Context, vm *vm.Instance, gameDir string, mods []ModConfig) error {
    // Download BepInEx
    bepinexPath := filepath.Join(gameDir, "BepInEx")
    if err := vm.Mkdir(ctx, bepinexPath, 0755); err != nil {
        return err
    }

    // Download and extract BepInEx
    bepinexURL := "https://github.com/BepInEx/BepInEx/releases/download/v5.4.23/BepInEx_unity_5.4.23.zip"
    if err := vm.Download(ctx, bepinexURL, bepinexPath+".zip"); err != nil {
        return fmt.Errorf("download bepinex: %w", err)
    }

    if err := vm.Exec(ctx, fmt.Sprintf("unzip -o %s.zip -d %s", bepinexPath+".zip", bepinexPath)); err != nil {
        return fmt.Errorf("extract bepinex: %w", err)
    }

    // Install mods
    for _, mod := range mods {
        if err := b.installMod(ctx, vm, bepinexPath, mod); err != nil {
            return fmt.Errorf("install mod %s: %w", mod.Name, err)
        }
    }

    // Create patch configs
    if err := b.applyPatches(ctx, vm, bepinexPath, mods); err != nil {
        return fmt.Errorf("apply patches: %w", err)
    }

    return nil
}

func (b *BepInExManager) installMod(ctx context.Context, vm *vm.Instance, bepinexPath string, mod ModConfig) error {
    modPath := filepath.Join(bepinexPath, "BepInEx", "plugins", mod.Name)
    if err := vm.Mkdir(ctx, modPath, 0755); err != nil {
        return err
    }

    // Download mod
    if err := vm.Download(ctx, mod.DownloadURL, modPath+".zip"); err != nil {
        return err
    }

    // Extract mod
    return vm.Exec(ctx, fmt.Sprintf("unzip -o %s.zip -d %s", modPath+".zip", modPath))
}
```

### Memory Scanner & Automation

```go
// pkg/game/automation/memory.go
package automation

import (
    "bytes"
    "context"
    "encoding/binary"
    "fmt"

    "github.com/KooshaPari/nanovms/pkg/vm"
)

type MemoryScanner struct {
    vm *vm.Instance
}

// ScanValue scans memory for a specific value
func (ms *MemoryScanner) ScanValue(ctx context.Context, value interface{}, valueType string) ([]uint64, error) {
    switch valueType {
    case "int32":
        return ms.scanInt32(ctx, value.(int32))
    case "float":
        return ms.scanFloat(ctx, value.(float32))
    case "string":
        return ms.scanString(ctx, value.(string))
    default:
        return nil, fmt.Errorf("unsupported type: %s", valueType)
    }
}

func (ms *MemoryScanner) scanInt32(ctx context.Context, target int32) ([]uint64, error) {
    data, err := ms.vm.ReadMemory(ctx, 0, ms.vm.MemorySize())
    if err != nil {
        return nil, err
    }

    var addrs []uint64
    for i := 0; i < len(data)-4; i++ {
        val := int32(binary.LittleEndian.Uint32(data[i:i+4]))
        if val == target {
            addrs = append(addrs, uint64(i))
        }
    }
    return addrs, nil
}

// PatchMemory patches memory at specific address
func (ms *MemoryScanner) PatchMemory(ctx context.Context, addr uint64, data []byte) error {
    return ms.vm.WriteMemory(ctx, addr, data)
}

// UnityECS integration
type UnityECSScanner struct {
    *MemoryScanner
    componentTypes map[string]uint64  // Type -> signature
}

// FindComponents finds Unity ECS components
func (s *UnityECSScanner) FindComponents(ctx context.Context, archetype string) ([]uint64, error) {
    // Find archetype by signature
    signature, ok := s.componentTypes[archetype]
    if !ok {
        return nil, fmt.Errorf("unknown archetype: %s", archetype)
    }

    // Scan for archetype signature
    return s.ScanValue(ctx, signature, "int32")
}

// ReadComponent reads Unity ECS component data
func (s *UnityECSScanner) ReadComponent(ctx context.Context, addr uint64, componentType string) (interface{}, error) {
    data, err := s.vm.ReadMemory(ctx, addr, 256)  // Max component size
    if err != nil {
        return nil, err
    }

    switch componentType {
    case "Transform":
        return s.parseTransform(data)
    case "Rigidbody":
        return s.parseRigidbody(data)
    case "Health":
        return s.parseHealth(data)
    default:
        return data, nil
    }
}
```

### Test Result Collection

```go
// pkg/game/results/results.go
package results

import (
    "encoding/xml"
    "fmt"
    "time"
)

type JUnitReport struct {
    XMLName   xml.Name   `xml:"testsuites"`
    Tests     int        `xml:"tests,attr"`
    Failures  int        `xml:"failures,attr"`
    Time      float64    `xml:"time,attr"`
    TestSuite []TestSuite `xml:"testsuite"`
}

type TestSuite struct {
    Name      string     `xml:"name,attr"`
    Tests     int        `xml:"tests,attr"`
    Failures  int        `xml:"failures,attr"`
    Time      float64    `xml:"time,attr"`
    TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
    Name      string     `xml:"name,attr"`
    ClassName string     `xml:"classname,attr"`
    Time      float64    `xml:"time,attr"`
    Failure   *Failure   `xml:"failure,omitempty"`
    SystemOut string     `xml:"system-out,omitempty"`
}

type Failure struct {
    Message string `xml:"message,attr"`
    Type    string `xml:"type,attr"`
    Content string `xml:",chardata"`
}

// Export exports results in multiple formats
func (r *ResultCollector) Export(format string) ([]byte, error) {
    switch format {
    case "junit":
        return r.exportJUnit()
    case "json":
        return r.exportJSON()
    case "allure":
        return r.exportAllure()
    case "prometheus":
        return r.exportPrometheus()
    default:
        return nil, fmt.Errorf("unknown format: %s", format)
    }
}

func (r *ResultCollector) exportJUnit() ([]byte, error) {
    report := JUnitReport{
        Tests:    len(r.results),
        Failures: r.failureCount,
        Time:     r.totalDuration.Seconds(),
        TestSuite: []TestSuite{{
            Name:  "GameAutomation",
            Tests: len(r.results),
            Failures: r.failureCount,
            Time: r.totalDuration.Seconds(),
            TestCases: r.junitTestCases(),
        }},
    }

    return xml.MarshalIndent(report, "", "  ")
}
```

### Performance Targets

| Metric | Target | Method |
|--------|--------|--------|
| **VM cold start** | <10s | Pre-copied compressed images + snapshots |
| **VM warm start** | <2s | Suspend-to-disk resume |
| **Game startup** | <15s | Pre-warmed with Steam token |
| **Test parallelism** | 8 per host | VM cluster with quota |
| **Memory per VM** | 4GB | Game + Steam + mods |
| **Storage per VM** | 20GB | Compressed base + mods |
| **Network** | localhost only | No external traffic |

### Consequences

### Positive
- Parallel test execution significantly reduces CI time
- Consistent environment via snapshots
- Steam token auth enables headless testing
- BepInEx support enables mod testing
- Memory scanner enables cheat detection testing

### Negative
- Steam API rate limits
- Game anti-cheat compatibility
- Mod conflicts
- Snapshot storage overhead
- VM licensing concerns
