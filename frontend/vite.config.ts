import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

/**
 * Vite config for the Lymuru Wails frontend.
 *
 * - Wails injects its runtime API in development mode; we don't need
 *   to proxy any HTTP traffic.
 * - Asset URLs like `/assets/image/...` are served by the embedded
 *   Wails asset server directly.
 */
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": "/src" },
  },
});
