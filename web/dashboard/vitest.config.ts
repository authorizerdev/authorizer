import { defineConfig } from 'vitest/config';

// Unit tests run in a plain Node environment — the suites here cover pure
// model/DSL logic, no DOM required. Kept separate from vite.config.ts so the
// production build config stays untouched.
export default defineConfig({
	test: {
		environment: 'node',
		include: ['src/**/*.test.ts'],
	},
});
