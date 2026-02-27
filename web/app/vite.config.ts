import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [react()],
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
					// Name all CSS files as index.css (since cssCodeSplit is false, there's only one)
					if (assetInfo.name && assetInfo.name.includes('.css')) {
						return 'index.css';
					}
					return 'assets/[name]-[hash][extname]';
				},
			},
		},
		cssCodeSplit: false, // Ensure CSS is in a single file
	},
});
