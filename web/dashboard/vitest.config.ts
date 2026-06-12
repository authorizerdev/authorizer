import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Unit tests default to a plain Node environment (pure model/DSL logic);
// component tests opt into jsdom per file via `// @vitest-environment jsdom`.
// Kept separate from vite.config.ts so the production build config stays
// untouched.
export default defineConfig({
	plugins: [react()],
	test: {
		environment: 'node',
		include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
	},
});
