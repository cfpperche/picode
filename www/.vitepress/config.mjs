import { defineConfig } from "vitepress";

export default defineConfig({
  title: "PiCode",
  description: "Browser-based Agent Development Environment for Pi coding agents",
  base: "/picode/",
  cleanUrls: true,
  lastUpdated: true,
  themeConfig: {
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
