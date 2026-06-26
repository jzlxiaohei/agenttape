import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// Served under /viewer in production (Go embeds/serves dist there).
export default defineConfig({
  base: "/viewer/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "src") },
  },
  // Build straight into the Go embed package so `go build` bundles the viewer into a
  // single self-contained binary (see internal/web). Only index.html is committed
  // there as a placeholder; this overwrites it with the real bundle.
  build: {
    outDir: path.resolve(__dirname, "../internal/web/dist"),
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      // Dev: proxy API calls to the running `agenttape serve`.
      "/api": "http://127.0.0.1:8787",
    },
  },
});
