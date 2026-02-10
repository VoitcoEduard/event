import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/events': {
        target: 'http://backend:8080',
        changeOrigin: true,
        secure: false,
      },
      '/templates': {
        target: 'http://backend:8080',
        changeOrigin: true,
        secure: false,
      }
    }
  }
})
