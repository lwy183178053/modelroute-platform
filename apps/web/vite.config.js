import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const proxyTarget = process.env.VITE_API_PROXY_TARGET;

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 57723,
    strictPort: true,
    proxy: proxyTarget ? {
      '/api': { target: proxyTarget, changeOrigin: true },
      '/v1': { target: proxyTarget, changeOrigin: true },
    } : undefined,
  },
});
