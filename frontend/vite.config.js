import { defineConfig, loadEnv } from 'vite';
import path from 'path';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const backendURL = env.VITE_BACKEND_URL || 'http://localhost:8080';

  return {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "@/styles/variables" as *;`,
        },
      },
    },
    server: {
      port: 5173,
      host: '0.0.0.0',
      proxy: {
        // Все вызовы /api/v1/* проксируются на backend.
        '/api': {
          target: backendURL,
          changeOrigin: true,
          // cookies проходят как есть, поскольку Set-Cookie приходит на тот же origin (5173 → 5173 для браузера)
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
    },
  };
});
