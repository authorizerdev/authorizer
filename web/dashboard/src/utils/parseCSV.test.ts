// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import parseCSV from './parseCSV';

// parseCSV wraps a FileReader in a Promise. The regression it guards against:
// a FileReader error (locked / unreadable file) used to leave the promise
// pending forever, so the "Import users" UI hung with no error surfaced.
describe('parseCSV', () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('resolves with parsed, validity-tagged entries on success', async () => {
		const file = new File(['a@b.com,not-an-email'], 'users.csv', {
			type: 'text/csv',
		});
		const result = await parseCSV(file, ',');
		expect(result).toEqual([
			{ value: 'a@b.com', isInvalid: false },
			{ value: 'not-an-email', isInvalid: true },
		]);
	});

	it('rejects (does not hang) when the FileReader errors', async () => {
		// Stub FileReader so readAsText drives the error path deterministically.
		class ErroringFileReader {
			onerror: (() => void) | null = null;
			onload: (() => void) | null = null;
			abort() {}
			readAsText() {
				this.onerror?.();
			}
		}
		vi.stubGlobal('FileReader', ErroringFileReader);

		const file = new File(['whatever'], 'users.csv', { type: 'text/csv' });
		await expect(parseCSV(file, ',')).rejects.toThrow(
			'Failed to read the selected file',
		);
	});
});
