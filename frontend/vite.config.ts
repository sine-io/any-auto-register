import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, __dirname, '')
  const pyProxyTarget = env.VITE_PY_PROXY_TARGET || 'http://127.0.0.1:8000'
  const goProxyTarget = env.VITE_GO_PROXY_TARGET || 'http://127.0.0.1:8080'

  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    build: {
      outDir: '../static',
      emptyOutDir: true,
    },
    server: {
      proxy: {
        '/api': pyProxyTarget,
        '/api-go': {
          target: goProxyTarget,
          rewrite: (path) => path.replace(/^\/api-go/, '/api'),
        },
      },
    },
  }
})
