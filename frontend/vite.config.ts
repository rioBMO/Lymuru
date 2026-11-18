import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
// wails.json is the single source of the app version: the Go side embeds it,
// and the release workflow rewrites productVersion from the git tag before building.
import wailsConfig from "../wails.json";

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
  define: {
    __APP_VERSION__: JSON.stringify(wailsConfig.info.productVersion),
  },
});
