import { defineConfig, type PluginOption } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [react()] as PluginOption[],
	build: {
		outDir: 'build',
		emptyOutDir: true,
		rollupOptions: {
			input: {
				main: 'src/index.tsx',
			},
			output: {
				entryFileNames: 'index.js',
				chunkFileNames: 'chunk-[name]-[hash].js',
				assetFileNames: (assetInfo) => {
					return 'assets/[name]-[hash][extname]';
				},
			},
		},
	},
	base: '/dashboard/',
	publicDir: 'public',
});
