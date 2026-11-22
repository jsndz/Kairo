import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import path from "path";

// https://vite.dev/config/
const target = process.env.KAIRO_API_TARGET ?? 'http://127.0.0.1:8080'
const proxy = {
  '/api': {
    target,
    ws: true,
  },
}

export default defineConfig({
  server: { proxy, strictPort: true },
  preview: { proxy },
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset()] }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
})
