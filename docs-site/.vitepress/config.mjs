import { defineConfig } from "vitepress";

export default defineConfig({
  title: "PiCode",
  description: "Browser-based Agent Development Environment for coding-agent CLIs",
  base: "/picode/",
  cleanUrls: true,
  lastUpdated: true,
  // Code blocks are dark in BOTH modes (Stripe/GitHub-docs pattern): the
  // block bg is #16161c even in the light theme, so Shiki must emit dark-
  // palette tokens or the highlighting disappears into the background.
  markdown: {
    theme: { light: "github-dark", dark: "github-dark" },
  },
  // Example URLs for the local app are not pages on this site. A bare
  // https://localhost:8445 in prose is a crawlable link and fails the
  // build — that froze GitHub Pages from 2026-08-29 until this config.
  ignoreDeadLinks: [/^https?:\/\/localhost/, /^https?:\/\/127\.0\.0\.1/],
  // Raw <video> tags in markdown carry their own asset URLs (the tutorial
  // MP4s live in public/, copied verbatim); Vite must not try to resolve
  // them as module imports.
  vue: { template: { transformAssetUrls: false } },
  head: [
    ["link", { rel: "icon", href: "/picode/favicon.svg", type: "image/svg+xml" }],
    ["link", { rel: "apple-touch-icon", href: "/picode/apple-touch-icon.png" }],
  ],
  themeConfig: {
    logo: "/favicon.svg",
    nav: [
      { text: "Start", link: "/guide/getting-started" },
      { text: "Use", link: "/guide/" },
      { text: "Changelog", link: "/changelog" },
      { text: "Commands", link: "/commands" },
    ],
    sidebar: [
      {
        // LibreChat audience split (docs/benchmarks/2026-09-14-librechat-docs.md):
        // Start = first run; Use = capabilities; Run = host it; Configure =
        // turn a thing on; Reference = lookup. Diátaxis modes still hold.
        text: "Start",
        items: [
          { text: "What is PiCode", link: "/" },
          { text: "Getting started", link: "/guide/getting-started" },
          { text: "From source", link: "/guide/from-source" },
          { text: "Development flow", link: "/guide/dev-flow" },
          { text: "Files and changes", link: "/guide/files" },
        ],
      },
      {
        text: "Use",
        link: "/guide/",
        collapsed: false,
        items: [
          {
            text: "Agents",
            collapsed: false,
            items: [
              { text: "Agent CLIs", link: "/guide/agent-clis" },
              { text: "Packages", link: "/guide/packages" },
              {
                text: "MCP",
                link: "/guide/mcp",
                collapsed: false,
                items: [
                  { text: "Gmail", link: "/guide/mcp-gmail" },
                  { text: "DeepWiki", link: "/guide/mcp-deepwiki" },
                ],
              },
              { text: "llama.cpp", link: "/guide/llama" },
              { text: "Automations", link: "/guide/automations" },
              { text: "Snippets", link: "/guide/snippets" },
            ],
          },
          {
            text: "Chat and work",
            collapsed: false,
            items: [
              { text: "Canvas", link: "/guide/canvas" },
              { text: "Session messages", link: "/guide/communication" },
              { text: "Delivery requests", link: "/guide/delivery" },
              { text: "Inbox tools", link: "/guide/inbox-tools" },
              { text: "Checklist", link: "/guide/checklist" },
              { text: "Pins and reminders", link: "/guide/pins" },
              { text: "Compact earlier", link: "/guide/compact" },
              { text: "Diff panel", link: "/guide/diff-panel" },
              { text: "Web apps", link: "/guide/web-apps" },
            ],
          },
          {
            text: "Tools",
            collapsed: false,
            items: [
              { text: "Browser tools for pi", link: "/guide/browser-tool" },
              { text: "Computer use for pi", link: "/guide/computer-tool" },
              { text: "PiCode tools for other agent CLIs", link: "/guide/picode-mcp" },
              { text: "Chrome extension", link: "/guide/browser-extension" },
              { text: "Docker and sysadmin", link: "/guide/docker" },
              { text: "Integrations", link: "/guide/integrations" },
              { text: "tmux sessions", link: "/guide/tmux" },
              { text: "CLI activity reporting", link: "/guide/terminal-status" },
              { text: "Dev servers", link: "/guide/dev-servers" },
              { text: "Keyboard and browser keys", link: "/guide/keyboard" },
            ],
          },
        ],
      },
      {
        text: "Run",
        collapsed: false,
        items: [
          { text: "On Windows", link: "/guide/windows-desktop" },
          { text: "Security and pairing", link: "/guide/security" },
          { text: "On a server", link: "/guide/remote-server" },
          { text: "Share one server", link: "/guide/shared-server" },
          { text: "Agent terminals over SSH", link: "/guide/ssh-terminals" },
          { text: "Open it to the internet", link: "/guide/public-access" },
          { text: "On your phone", link: "/guide/mobile" },
        ],
      },
      {
        text: "Configure",
        link: "/guide/configure",
        collapsed: false,
        items: [
          { text: "Settings", link: "/guide/settings" },
          { text: "Providers", link: "/guide/providers" },
          { text: "Model roles", link: "/guide/roles" },
          { text: "Backup and restore", link: "/guide/backup" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "Changelog", link: "/changelog" },
          { text: "Commands", link: "/commands" },
          { text: "HTTP API", link: "/api" },
          { text: "License", link: "/license" },
        ],
      },
    ],
    search: { provider: "local" },
    outline: "deep",
    editLink: {
      pattern:
        "https://github.com/cfpperche/picode/edit/main/docs-site/:path",
      text: "Edit this page on GitHub",
    },
    socialLinks: [
      { icon: "github", link: "https://github.com/cfpperche/picode" },
    ],
  },
});
