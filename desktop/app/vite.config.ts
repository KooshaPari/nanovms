import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Vite config for AgilePlus Desktop — Electrobun webview
// In dev mode the dev server runs on port 1421 which Electrobun loads
// directly. In production the built assets are loaded from disk.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 1421,
    strictPort: true,
    // Allow Electrobun webview to connect
    headers: {
      "Access-Control-Allow-Origin": "*",
    },
  },
  build: {
    outDir: "dist",
    sourcemap: true,
    // Emit to desktop/electrobun/dist so Electrobun can package it
    rollupOptions: {
      output: {
        dir: "../electrobun/dist/web",
      },
    },
  },
  // Resolve paths for the hook imports
  resolve: {
    alias: {
      "@": "/src",
    },
  },
});
