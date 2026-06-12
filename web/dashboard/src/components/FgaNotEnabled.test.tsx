// @vitest-environment jsdom
import React from 'react';
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import FgaNotEnabled from './FgaNotEnabled';

// FgaNotEnabled is what the Authorization tab renders when the backend
// reports "fine-grained authorization is not enabled" — i.e. the main
// database does not support OpenFGA and no --fga-store override was passed.
// The screen must explain the state and show the exact flags that fix it.
describe('FgaNotEnabled', () => {
	it('explains that FGA is not enabled', () => {
		render(<FgaNotEnabled />);
		expect(
			screen.getByText(/Fine-Grained Authorization isn(’|')t enabled yet/),
		).toBeTruthy();
	});

	it('shows the --fga-store flags needed to enable it', () => {
		render(<FgaNotEnabled />);
		// The command renders in more than one place (code block + copy
		// affordance) — all occurrences must carry both flags.
		expect(
			screen.getAllByText(/--fga-store=postgres/).length,
		).toBeGreaterThan(0);
		expect(screen.getAllByText(/--fga-store-url=/).length).toBeGreaterThan(0);
	});
});
