import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [react()],
	server: {
		// In dev, the React app is served by Vite (e.g. :5173) while the
		// Authorizer backend serves OAuth/OIDC endpoints (default :8080).
		// Proxy auth endpoints so `/authorize` and friends work on the same
		// origin when using tunnels (loca.lt) or local testing.
		proxy: {
			'^/(authorize|oauth|userinfo|jwks|logout|healthz)(/.*)?$': {
				target: 'http://localhost:8080',
				changeOrigin: true,
			},
			'^/\\.well-known(/.*)?$': {
				target: 'http://localhost:8080',
				changeOrigin: true,
			},
			'^/graphql$': {
				target: 'http://localhost:8080',
				changeOrigin: true,
			},
		},
	},
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
