import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "NanoVMS",
  description: "SOTA Virtualization for Cloud Infrastructure on Consumer Hardware",

  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      {
        text: 'Guides',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Architecture', link: '/guide/architecture' },
          { text: 'VM Flavors', link: '/guide/vm-flavors' },
          { text: 'Sandbox Isolation', link: '/guide/sandbox-isolation' },
          { text: 'GPU Passthrough', link: '/guide/gpu-passthrough' },
          { text: 'Game Automation', link: '/guide/game-automation' },
          { text: 'Agent Workloads', link: '/guide/agent-workloads' },
        ]
      },
      {
        text: 'Journeys',
        items: [
          { text: 'Overview', link: '/journeys/' },
          { text: 'Quick Start', link: '/journeys/quick-start' },
          { text: 'Game Automation', link: '/journeys/game-automation' },
          { text: 'Agent Desktop', link: '/journeys/agent-desktop' },
          { text: 'GPU Passthrough', link: '/journeys/gpu-passthrough' },
        ]
      },
      {
        text: 'Stories',
        items: [
          { text: 'Overview', link: '/stories/' },
          { text: 'Create First VM', link: '/stories/create-first-vm' },
          { text: 'Run Game Tests', link: '/stories/run-game-tests' },
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI Reference', link: '/reference/cli' },
          { text: 'Configuration', link: '/reference/configuration' },
          { text: 'API Reference', link: '/reference/api' },
          { text: 'Architecture', link: '/reference/architecture' },
          { text: 'Benchmarks', link: '/reference/benchmarks' },
        ]
      },
      {
        text: 'Research',
        items: [
          { text: 'VFIO/GPU Passthrough', link: '/research/vfio' },
          { text: 'Firecracker Deep Dive', link: '/research/firecracker' },
          { text: 'io_uring Performance', link: '/research/iouring' },
          { text: 'eBPF Networking', link: '/research/ebpf' },
          { text: 'SOTA Comparison', link: '/research/comparison' },
          { text: 'Hardware Profiles', link: '/research/hardware-profiles' },
        ]
      },
      { text: 'Changelog', link: '/changelog' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/guide/getting-started' },
            { text: 'Quick Start', link: '/guide/quickstart' },
            { text: 'Configuration', link: '/guide/configuration' },
          ]
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Architecture Overview', link: '/guide/architecture' },
            { text: 'VM Flavors', link: '/guide/vm-flavors' },
            { text: 'Sandbox Isolation', link: '/guide/sandbox-isolation' },
            { text: 'Storage', link: '/guide/storage' },
          ]
        },
        {
          text: 'Use Cases',
          items: [
            { text: 'GPU Passthrough', link: '/guide/gpu-passthrough' },
            { text: 'Game Automation', link: '/guide/game-automation' },
            { text: 'Agent Workloads', link: '/guide/agent-workloads' },
            { text: 'Development Environments', link: '/guide/dev-environments' },
          ]
        },
        {
          text: 'Advanced',
          items: [
            { text: 'Performance Tuning', link: '/guide/performance' },
            { text: 'Security Hardening', link: '/guide/security' },
            { text: 'Troubleshooting', link: '/guide/troubleshooting' },
          ]
        },
      ],
      '/journeys/': [
        {
          text: 'User Journeys',
          items: [
            { text: 'Overview', link: '/journeys/' },
            { text: 'Quick Start Journey', link: '/journeys/quick-start' },
            { text: 'Game Automation Journey', link: '/journeys/game-automation' },
            { text: 'Agent Desktop Journey', link: '/journeys/agent-desktop' },
            { text: 'GPU Passthrough Journey', link: '/journeys/gpu-passthrough' },
          ]
        },
      ],
      '/stories/': [
        {
          text: 'User Stories',
          items: [
            { text: 'Overview', link: '/stories/' },
            { text: 'Create Your First VM', link: '/stories/create-first-vm' },
            { text: 'Run Game Tests', link: '/stories/run-game-tests' },
          ]
        },
      ],
      '/reference/': [
        {
          text: 'CLI Reference',
          items: [
            { text: 'Command Overview', link: '/reference/cli' },
            { text: 'vm Commands', link: '/reference/cli-vm' },
            { text: 'sandbox Commands', link: '/reference/cli-sandbox' },
            { text: 'game Commands', link: '/reference/cli-game' },
            { text: 'agent Commands', link: '/reference/cli-agent' },
            { text: 'vfio Commands', link: '/reference/cli-vfio' },
          ]
        },
        {
          text: 'Configuration',
          items: [
            { text: 'Config File', link: '/reference/configuration' },
            { text: 'Profiles', link: '/reference/profiles' },
            { text: 'Environment Variables', link: '/reference/environment' },
          ]
        },
        {
          text: 'API Reference',
          items: [
            { text: 'REST API', link: '/reference/api' },
            { text: 'gRPC API', link: '/reference/grpc' },
            { text: 'WebSocket API', link: '/reference/websocket' },
          ]
        },
      ],
      '/adr/': [
        {
          text: 'Architecture Decisions',
          items: [
            { text: 'ADR Index', link: '/adr/' },
            { text: 'ADR-001: Language Selection', link: '/adr/ADR-001-language-selection' },
            { text: 'ADR-002: Three-Tier Isolation', link: '/adr/ADR-002-three-tier-isolation' },
            { text: 'ADR-003: Storage Backend', link: '/adr/ADR-003-storage-backend' },
            { text: 'ADR-004: Networking Model', link: '/adr/ADR-004-networking-model' },
            { text: 'ADR-005: Snapshot Strategy', link: '/adr/ADR-005-snapshot-strategy' },
            { text: 'ADR-006: Metrics & Observability', link: '/adr/ADR-006-observability' },
          ]
        },
      ],
    },

    search: {
      provider: 'local'
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/KooshaPari/nanovms' }
    ]
  }
})
