import { defineConfig } from "vitepress";

export default defineConfig({
  title: "PiCode",
  description: "Browser-based Agent Development Environment for Pi coding agents",
  base: "/picode/",
  cleanUrls: true,
  lastUpdated: true,
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
        text: "Start",
        items: [
          { text: "What is PiCode", link: "/" },
          { text: "Getting started", link: "/guide/getting-started" },
          { text: "Providers", link: "/guide/providers" },
          { text: "Packages", link: "/guide/packages" },
          { text: "MCP", link: "/guide/mcp" },
          { text: "llama.cpp", link: "/guide/llama" },
          { text: "Chrome extension", link: "/guide/browser-extension" },
          { text: "On your phone", link: "/guide/mobile" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "Commands", link: "/commands" },
          { text: "Settings", link: "/guide/settings" },
          { text: "License", link: "/license" },
        ],
      },
    ],
    search: { provider: "local" },
    outline: "deep",
    socialLinks: [
      { icon: "github", link: "https://github.com/cfpperche/picode" },
    ],
  },
});
