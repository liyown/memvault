import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Memvault',
  description: 'Local-first conversation archive and memory layer for AI agents',
  base: '/memvault/',
  ignoreDeadLinks: true,

  head: [
    ['link', { rel: 'icon', href: '/memvault/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#3b6dff' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:locale', content: 'en' }],
    ['meta', { name: 'og:site_name', content: 'Memvault' }],
    ['meta', { name: 'og:image', content: 'https://liyown.github.io/memvault/hero-illustration.svg' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:image', content: 'https://liyown.github.io/memvault/hero-illustration.svg' }],
  ],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Concepts', link: '/concepts/conversation-vs-memory' },
      { text: 'GitHub', link: 'https://github.com/liyown/memvault' }
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'What is Memvault?', link: '/guide/introduction' },
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Installation', link: '/guide/installation' }
        ]
      },
      {
        text: 'Core Concepts',
        items: [
          { text: 'Conversation vs Memory', link: '/concepts/conversation-vs-memory' },
          { text: 'Why Local-First?', link: '/concepts/why-local-first' },
          { text: 'Data Sovereignty', link: '/concepts/data-sovereignty' },
          { text: 'MCP Integration', link: '/concepts/mcp' },
          { text: 'Storage Design', link: '/concepts/storage' }
        ]
      },
      {
        text: 'Guides',
        items: [
          { text: 'Archiving Conversations', link: '/guide/archiving' },
          { text: 'Full-Text Search', link: '/guide/search' },
          { text: 'Agent Memory', link: '/guide/memory' },
          { text: 'Multi-Device Sync', link: '/guide/sync' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI Commands', link: '/reference/cli' },
          { text: 'MCP Tools', link: '/reference/mcp-tools' },
          { text: 'Configuration', link: '/reference/configuration' },
          { text: 'API Reference', link: '/reference/api' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/liyown/memvault' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026-present Memvault Contributors'
    },

    search: {
      provider: 'local'
    }
  }
})
