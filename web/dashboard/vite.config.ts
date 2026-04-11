import { defineConfig, type PluginOption } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [react(), tailwindcss()] as PluginOption[],
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
					if (assetInfo.names?.some((name) => name.endsWith('.css'))) {
						return '[name][extname]';
					}
					return 'assets/[name]-[hash][extname]';
				},
			},
		},
	},
	base: '/dashboard/build/',
	publicDir: 'public',
});
