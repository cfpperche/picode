import { defineConfig } from "vitepress";

export default defineConfig({
  title: "PiCode",
  description: "Browser-based Agent Development Environment for Pi coding agents",
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
  head: [
    ["link", { rel: "icon", href: "/picode/favicon.svg", type: "image/svg+xml" }],
    ["link", { rel: "apple-touch-icon", href: "/picode/apple-touch-icon.png" }],
  ],
  themeConfig: {
    logo: "/favicon.svg",
    nav: [
      { text: "Guide", link: "/guide/getting-started" },
      { text: "Commands", link: "/commands" },
    ],
    sidebar: [
      {
        // Diátaxis quadrants (docs/benchmarks.md, Documentation benchmarks):
        // Start teaches by doing; Guides answer "how do I…"; Run it somewhere
        // is the self-hosting how-tos; Reference is lookup, not learning.
        text: "Start",
        items: [
          { text: "What is PiCode", link: "/" },
          { text: "Getting started", link: "/guide/getting-started" },
        ],
      },
      {
        text: "Guides",
        items: [
          { text: "Providers", link: "/guide/providers" },
          { text: "Packages", link: "/guide/packages" },
          { text: "Checklist", link: "/guide/checklist" },
          { text: "MCP", link: "/guide/mcp" },
          { text: "Terminal status for CLIs", link: "/guide/terminal-status" },
          { text: "llama.cpp", link: "/guide/llama" },
          { text: "Chrome extension", link: "/guide/browser-extension" },
          { text: "Automations", link: "/guide/automations" },
        ],
      },
      {
        text: "Run it somewhere",
        items: [
          { text: "Security and pairing", link: "/guide/security" },
          { text: "On a server", link: "/guide/remote-server" },
          { text: "Share one server", link: "/guide/shared-server" },
          { text: "Open it to the internet", link: "/guide/public-access" },
          { text: "On your phone", link: "/guide/mobile" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "Commands", link: "/commands" },
          { text: "Settings", link: "/guide/settings" },
          { text: "HTTP API", link: "/api" },
          { text: "License", link: "/license" },
        ],
      },
    ],
    search: { provider: "local" },
    outline: "deep",
    editLink: {
      pattern:
        "https://github.com/cfpperche/picode/edit/main/www/:path",
      text: "Edit this page on GitHub",
    },
    socialLinks: [
      { icon: "github", link: "https://github.com/cfpperche/picode" },
    ],
  },
});
