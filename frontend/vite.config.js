import { defineConfig, loadEnv } from 'vite';
import path from 'path';

export default defineConfig(({ mode }) => {
  // .env лежит в корне проекта (на уровень выше frontend/), а не в frontend/.
  // Vite по умолчанию читает из cwd → явно указываем envDir = корень.
  const envDir = path.resolve(__dirname, '..');
  const env = loadEnv(mode, envDir, '');
  const backendURL = env.VITE_BACKEND_URL || 'http://localhost:8080';

  return {
    envDir,
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
