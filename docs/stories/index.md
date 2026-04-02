---
head:
  - - meta
    - name: description
      content: "User stories demonstrating NanoVMS capabilities with real-world examples"
---

# User Stories

> Real-world NanoVMS use cases with step-by-step examples

Each story demonstrates a complete user workflow with code examples, GIFs, and expected outcomes.

## Featured Stories

<StoryCardGrid>
  <StoryCard
    title="Create Your First VM"
    description="5-minute quick start guide for creating and running your first NanoVMS VM"
    link="./create-first-vm"
    image="/images/stories/first-vm.gif"
    tags={['beginner', 'quickstart', 'firecracker']}
  />
  <StoryCard
    title="Run Game Automation Tests"
    description="Parallel game testing with Steam headless and automated UI interactions"
    link="./run-game-tests"
    image="/images/stories/game-tests.gif"
    tags={['gaming', 'automation', 'ci/cd']}
  />
  <StoryCard
    title="Setup GPU Passthrough"
    description="Configure VFIO GPU passthrough for near-bare-metal performance"
    link="./setup-gpu-passthrough"
    image="/images/stories/gpu-passthrough.gif"
    tags={['advanced', 'vfio', 'performance']}
  />
  <StoryCard
    title="Agent Desktop Environment"
    description="Create isolated desktop environments for AI agents with full GUI access"
    link="./agent-desktop"
    image="/images/stories/agent-desktop.gif"
    tags={['agents', 'gui', 'sandbox']}
  />
</StoryCardGrid>

## Story Categories

### Getting Started
- [Create Your First VM](./create-first-vm.md)
- [Understanding VM Flavors](./vm-flavors-explained.md)
- [Configuration Basics](./config-basics.md)

## Contributing Stories

Have a NanoVMS use case to share? See [Contributing Stories](../contributing.md#stories).
