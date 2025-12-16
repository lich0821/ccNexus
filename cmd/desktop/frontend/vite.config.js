import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    port: 34115,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
