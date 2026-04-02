import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Devenv Abstraction',
  description: 'Docker-alternative VM stack with OCI/sandbox support',
  srcDir: '.',
  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }]
  ],
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/' },
      { text: 'Reference', link: '/reference/' },
      { text: 'GitHub', link: 'https://github.com/KooshaPari/devenv-abstraction' }
    ],
    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Introduction', link: '/guide/' },
          { text: 'Architecture', link: '/guide/architecture' },
          { text: 'Adapters', link: '/guide/adapters' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI', link: '/reference/cli' },
          { text: 'API', link: '/reference/api' }
        ]
      }
    ]
  },
  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  }
})
