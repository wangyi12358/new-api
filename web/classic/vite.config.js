/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import react from '@vitejs/plugin-react';
import { defineConfig, transformWithEsbuild } from 'vite';
import pkg from '@douyinfe/vite-plugin-semi';
import path from 'path';
import { codeInspectorPlugin } from 'code-inspector-plugin';

const { vitePluginSemi } = pkg;

// https://vitejs.dev/config/
export default defineConfig(({ command }) => {
  const isBuild = command === 'build';

  return {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    plugins: [
      // Dev-only: avoid shipping inspector hooks and extra transforms in production builds.
      !isBuild &&
        codeInspectorPlugin({
          bundler: 'vite',
        }),
      {
        name: 'treat-js-files-as-jsx',
        async transform(code, id) {
          if (!/src\/.*\.js$/.test(id)) {
            return null;
          }

          return transformWithEsbuild(code, id, {
            loader: 'jsx',
            jsx: 'automatic',
          });
        },
      },
      // .js files are pre-transformed above; only run React plugin on .jsx to avoid double passes.
      react({
        include: /\.jsx$/,
      }),
      vitePluginSemi({
        cssLayer: true,
      }),
    ].filter(Boolean),
    esbuild: {
      jsx: 'automatic',
      legalComments: 'none',
    },
    optimizeDeps: {
      // force: true re-scans deps on every dev start; only needed when debugging dep cache.
      esbuildOptions: {
        loader: {
          '.js': 'jsx',
          '.json': 'json',
        },
      },
    },
    build: {
      target: 'es2020',
      // Skip gzip size calculation — saves noticeable time on large chunks (mermaid, semi-ui, etc.).
      reportCompressedSize: false,
      rollupOptions: {
        output: {
          manualChunks: {
            'react-core': ['react', 'react-dom', 'react-router-dom'],
            'semi-ui': ['@douyinfe/semi-icons', '@douyinfe/semi-ui'],
            tools: ['axios', 'history', 'marked'],
            'react-components': [
              'react-dropzone',
              'react-fireworks',
              'react-telegram-login',
              'react-toastify',
              'react-turnstile',
            ],
            i18n: [
              'i18next',
              'react-i18next',
              'i18next-browser-languagedetector',
            ],
            // Split heavy vendors so minification can run in parallel on CI/low-core servers.
            mermaid: ['mermaid'],
            vchart: [
              '@visactor/vchart',
              '@visactor/react-vchart',
              '@visactor/vchart-semi-theme',
            ],
            markdown: [
              'react-markdown',
              'remark-gfm',
              'remark-math',
              'remark-breaks',
              'rehype-highlight',
              'rehype-katex',
              'katex',
            ],
            icons: ['@lobehub/icons', 'lucide-react', 'react-icons'],
          },
        },
      },
    },
    server: {
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: 'https://v2.kovar.ai',
          // target: 'http://127.0.0.1:3000',
          changeOrigin: true,
        },
        '/mj': {
          target: 'https://v2.kovar.ai',
          // target: 'http://127.0.0.1:3000',
          changeOrigin: true,
        },
        '/pg': {
          target: 'https://v2.kovar.ai',
          // target: 'http://127.0.0.1:3000',
          changeOrigin: true,
        },
      },
    },
  };
});
