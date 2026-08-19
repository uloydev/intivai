import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": path.resolve(import.meta.dirname, "./src") } },
  server: {
    proxy: {
      "/api": { target: "http://localhost:8081", changeOrigin: true, ws: true },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/react/') || id.includes('node_modules/react-dom/') || id.includes('node_modules/react-router')) {
            return 'vendor';
          }
          if (id.includes('node_modules/@tanstack/')) {
            return 'query';
          }
          if (id.includes('node_modules/@phosphor-icons/') || id.includes('node_modules/lucide-react/')) {
            return 'icons';
          }
        },
      },
    },
  },
})
