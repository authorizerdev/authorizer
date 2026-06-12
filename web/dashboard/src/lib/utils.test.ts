import { describe, expect, it } from 'vitest';
import { isFgaNotEnabledError } from './utils';

// isFgaNotEnabledError is the single decision point that switches the
// Authorization tab (and the user permissions modal) into the FgaNotEnabled
// state — e.g. when the main database does not support OpenFGA and no
// --fga-store override is configured.
describe('isFgaNotEnabledError', () => {
	it('detects the backend not-enabled error', () => {
		expect(
			isFgaNotEnabledError({
				message: '[GraphQL] fine-grained authorization is not enabled',
			}),
		).toBe(true);
	});

	it('is case-insensitive', () => {
		expect(
			isFgaNotEnabledError({
				message: 'Fine-Grained Authorization Is Not Enabled',
			}),
		).toBe(true);
	});

	it('ignores unrelated errors', () => {
		expect(isFgaNotEnabledError({ message: 'unauthorized' })).toBe(false);
		expect(
			isFgaNotEnabledError({ message: 'authorization check failed' }),
		).toBe(false);
	});

	it('handles missing errors safely', () => {
		expect(isFgaNotEnabledError(undefined)).toBe(false);
		expect(isFgaNotEnabledError(null)).toBe(false);
		expect(isFgaNotEnabledError({})).toBe(false);
	});
});
