import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

const buildVersion = process.env.APP_BUILD_VERSION || Date.now().toString(36);

export default defineConfig({
  define: {
    __APP_BUILD_VERSION__: JSON.stringify(buildVersion),
  },
  plugins: [
    vue(),
    {
      name: 'app-version-manifest',
      generateBundle() {
        this.emitFile({
          type: 'asset',
          fileName: 'version.json',
          source: `${JSON.stringify({ version: buildVersion })}\n`,
        });
      },
    },
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
});
