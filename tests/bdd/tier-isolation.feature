Feature: Three-Tier Isolation Architecture
  As an AI agent developer
  I want reliable isolation tiers for running untrusted code
  So that I can safely execute LLM-generated tools and scripts

  Background:
    Given NanoVMS is installed and configured
    And the sandbox CLI is available at "./nanovms"

  # ============================================================================
  # Tier 1: WASM Sandboxes (Trusted Tools)
  # ============================================================================
  @tier-1 @wasm @fast @trusted
  Scenario: Execute trusted tool in WASM sandbox
    Given I have a trusted WASM module "formatter.wasm"
    When I create a sandbox with tier "wasm"
    And I execute "format --input code.rs" in the sandbox
    Then the sandbox should start in less than "5" milliseconds
    And the memory usage should be less than "2" MB
    And the output should be formatted code
    And no syscalls should escape the WASI sandbox

  @tier-1 @wasm @trusted
  Scenario: WASM sandbox resource limits
    Given I have a WASM module that allocates memory
    When I run it with memory limit "10MB"
    Then it should be terminated if memory exceeds "10MB"
    And I should see error "Memory limit exceeded"

  @tier-1 @wasm @trusted @fr-tdd-001
  Scenario: FR-TDD-001 - Fast tool execution
    Given the requirement "FR-TDD-001: Tool startup < 1ms"
    When I execute "nanovms sandbox create --tier wasm"
    Then the cold start time should be less than "1" ms
    And the warm start time should be less than "0.5" ms

  # ============================================================================
  # Tier 2: gVisor Containers (Semi-Trusted)
  # ============================================================================
  @tier-2 @gvisor @syscall-filter @semi-trusted
  Scenario: Execute third-party script in gVisor container
    Given I have a Node.js script from npm
    When I create a sandbox with tier "gvisor"
    And I run "npm install && node script.js"
    Then the container should start in less than "100" milliseconds
    And network access should be filtered by default
    And filesystem access should be restricted to workdir
    And syscall "execve" should be logged

  @tier-2 @gvisor @security @fr-tdd-002
  Scenario: FR-TDD-002 - Syscall interception
    Given the requirement "FR-TDD-002: Userspace kernel isolation"
    When a sandboxed process calls "open('/etc/passwd', O_RDONLY)"
    Then gVisor Sentry should intercept the syscall
    And the call should be denied with "Permission denied"
    And the violation should be logged

  @tier-2 @gvisor @network @semi-trusted
  Scenario: Network isolation in gVisor
    Given a running gVisor container
    When the process tries to connect to "8.8.8.8:53"
    Then the connection should be blocked by default
    When I whitelist "8.8.8.8/32" port "53"
    Then the DNS query should succeed

  # ============================================================================
  # Tier 3: Firecracker MicroVMs (Untrusted)
  # ============================================================================
  @tier-3 @firecracker @microvm @untrusted @full-isolation
  Scenario: Execute untrusted Docker image in MicroVM
    Given I have a Docker image from untrusted registry
    When I create a MicroVM with "firecracker" flavor
    And I pull and run the image
    Then the VM should start in less than "125" milliseconds
    And the VM memory should be less than "5" MB overhead
    And the image should run in full hardware isolation
    And VT-x/AMD-V should protect the host

  @tier-3 @firecracker @security @fr-tdd-003
  Scenario: FR-TDD-003 - Hardware isolation for LLM code
    Given the requirement "FR-TDD-003: Full VM isolation for generated code"
    When I execute arbitrary LLM-generated binary
    Then it should run in Firecracker MicroVM
    And host kernel should be protected by KVM
    And only 5 virtio devices should be exposed

  @tier-3 @firecracker @oci @untrusted
  Scenario: OCI image compatibility in MicroVM
    Given an OCI-compliant container image
    When I execute "nanovms vm create --flavor microvm --image alpine:latest"
    Then the image should boot successfully
    And OCI runtime spec should be honored
    And rootfs should be mounted from container layers

  # ============================================================================
  # Tier Selection and Escalation
  # ============================================================================
  @tier-selection @trust-level
  Scenario Outline: Automatic tier selection based on trust
    Given code with trust level "<trust_level>"
    When I execute "nanovms sandbox create --auto-tier"
    Then the tier should be "<selected_tier>"

    Examples:
      | trust_level    | selected_tier |
      | agent-native   | wasm          |
      | first-party    | wasm          |
      | third-party    | gvisor        |
      | llm-generated  | microvm       |
      | untrusted      | microvm       |

  @tier-escalation @security
  Scenario: Tier escalation on security violation
    Given a running Tier 2 (gVisor) sandbox
    When the process attempts "ptrace(PTRACE_ATTACH)"
    Then the syscall should be blocked
    And the system should escalate to Tier 3 for subsequent runs
    And a security alert should be logged

  # ============================================================================
  # Performance Benchmarks (Aligned with ADR-001)
  # ============================================================================
  @benchmark @performance @adr-001
  Scenario Outline: Tier performance benchmarks
    Given NanoVMS performance test harness
    When I measure tier "<tier>" startup
    Then it should complete in less than "<max_time>" ms
    And memory overhead should be less than "<max_memory>" MB

    Examples:
      | tier    | max_time | max_memory | source                    |
      | wasm    | 1        | 1          | Wasmtime benchmark      |
      | gvisor  | 90       | 20         | gVisor runsc benchmark  |
      | microvm | 125      | 5          | Firecracker benchmark   |

  @benchmark @comparison
  Scenario: Tier comparison for agent workloads
    Given 1000 agent tool invocations
    When I run with tier "wasm"
    Then total overhead should be less than "1" second
    When I run with tier "gvisor"
    Then total overhead should be less than "90" seconds
    When I run with tier "microvm"
    Then total overhead should be less than "125" seconds

  # ============================================================================
  # Error Handling and Recovery
  # ============================================================================
  @error-handling @resilience
  Scenario: Graceful degradation when tier unavailable
    Given Firecracker is not installed
    When I request tier "microvm"
    Then I should see warning "Firecracker not available"
    And the system should fallback to tier "gvisor"
    And the workload should still execute safely

  @error-handling @cleanup
  Scenario: Sandbox cleanup on failure
    Given a running sandbox with ID "test-123"
    When the process crashes with SIGSEGV
    Then the sandbox should be automatically cleaned up
    And resources should be freed within "5" seconds
    And no zombie processes should remain

  # ============================================================================
  # Integration with AI Agent Workflow
  # ============================================================================
  @ai-agent @integration @claude-code
  Scenario: Claude Code tool execution via NanoVMS
    Given Claude Code wants to run "cargo fmt"
    When it invokes "nanovms sandbox exec --tier wasm cargo fmt"
    Then the formatter should run in WASM sandbox
    And the formatted code should be returned
    And execution time should be minimal

  @ai-agent @integration @security
  Scenario: LLM-generated code execution
    Given LLM generates Python script to analyze data
    When the script is executed in Tier 3 MicroVM
    Then it should run with full isolation
    And if malicious, the host should remain secure
    And output should be captured and returned
