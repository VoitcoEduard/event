import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const backend = process.env.VITE_BACKEND_TARGET || 'http://localhost:8080'
const paths = ['/auth', '/events', '/sessions', '/screensavers', '/tracks', '/media', '/templates']

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
	proxy: Object.fromEntries([
		...paths.map(path => [path, { target: backend, changeOrigin: true }]),
		['/ws', { target: backend, ws: true, changeOrigin: true }]
	])
  }
})
