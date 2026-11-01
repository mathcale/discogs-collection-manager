import { defineConfig } from 'vite';
import { minifyHtmlLiteralsPlugin } from './config/base';

export default defineConfig({
  plugins: [minifyHtmlLiteralsPlugin()],
  build: {
    outDir: 'out',
    emptyOutDir: true,
    lib: {
      entry: 'src/js/pin-entry.ts',
      name: '__Z_PIN__',
      fileName: 'pin',
      formats: ['iife'],
    },
  },
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  esbuild: { legalComments: 'none' },
});
