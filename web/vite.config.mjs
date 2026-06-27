import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { defineConfig } from "vite";

const apiTarget = process.env.HARNESS_UI_API_TARGET ?? "http://127.0.0.1:4310";
const configDir = fileURLToPath(new URL(".", import.meta.url));
const outDir = resolve(configDir, "../internal/ui/generated/build");

export default defineConfig({
  build: {
    outDir,
    emptyOutDir: true,
  },
  server: {
    host: "127.0.0.1",
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: "127.0.0.1",
  },
});
