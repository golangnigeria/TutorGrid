import { defineConfig } from "vite";
import path from "path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      "/auth": "http://localhost:8080",
      "/api": "http://localhost:8080",
      "/status": "http://localhost:8080",
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@boilerplate/openapi": path.resolve(
        __dirname,
        "../../packages/openapi/src"
      ),
      "@boilerplate/zod": path.resolve(__dirname, "../../packages/zod/src"),
    },
  },
});
