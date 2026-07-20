// @vitest-environment jsdom
import React from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import {
	render,
	screen,
	fireEvent,
	waitFor,
	cleanup,
} from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Tuples from './Tuples';
import { FgaGetModelQuery, FgaReadTuplesQuery } from '../../graphql/queries';

// urql's useClient is the only external dependency Tuples.tsx needs.
vi.mock('urql', () => ({
	useClient: () => mockClient,
}));

let mockClient: {
	query: ReturnType<typeof vi.fn>;
	mutation: ReturnType<typeof vi.fn>;
};

const modelExistsResponse = {
	data: { _fga_get_model: { dsl: 'model\n  schema 1.1' } },
};

const emptyTuplesResponse = {
	data: { _fga_read_tuples: { tuples: [], continuation_token: '' } },
};

const oneTupleResponse = {
	data: {
		_fga_read_tuples: {
			tuples: [{ user: 'user:1', relation: 'viewer', object: 'document:1' }],
			continuation_token: '',
		},
	},
};

function renderTuples() {
	return render(
		<MemoryRouter>
			<Tuples />
		</MemoryRouter>,
	);
}

// This project's vitest.config.ts does not set `test.globals: true`, so
// @testing-library/react's automatic afterEach(cleanup) never registers —
// without this, a Radix Dialog portal left open at the end of one test (e.g.
// the confirm dialog before its close animation/state settles) leaks into
// the next test's DOM and causes bogus "multiple elements found" failures.
afterEach(cleanup);

beforeEach(() => {
	mockClient = {
		query: vi.fn(() => ({
			toPromise: () => Promise.resolve(emptyTuplesResponse),
		})),
		mutation: vi.fn(() => ({
			toPromise: () =>
				Promise.resolve({ data: { _fga_write_tuples: { message: 'ok' } } }),
		})),
	};
});

// Route each client.query() call by which document it's asking for, not by
// call order — the sibling <AuthSteps> component fires its own
// FgaGetModel/FgaReadTuples queries (for the step-completion checkmarks) in
// the same render, so call count/order isn't deterministic across components.
function mockQueries(tuplesResponse: unknown) {
	mockClient.query.mockImplementation((doc: unknown) => {
		const response =
			doc === FgaReadTuplesQuery ? tuplesResponse : modelExistsResponse;
		return { toPromise: () => Promise.resolve(response) };
	});
}

describe('Tuples page — confirmation before mutating', () => {
	it('opens a confirm dialog on submit and only fires the mutation after confirming', async () => {
		mockQueries(emptyTuplesResponse);
		renderTuples();

		await screen.findByText('No relationship tuples yet');

		fireEvent.change(screen.getByLabelText('User'), {
			target: { value: 'user:42' },
		});
		fireEvent.change(screen.getByLabelText('Relation'), {
			target: { value: 'viewer' },
		});
		fireEvent.change(screen.getByLabelText('Object'), {
			target: { value: 'document:9' },
		});
		fireEvent.click(screen.getByRole('button', { name: /^Add$/ }));

		// Dialog appears; mutation has NOT fired yet.
		expect(await screen.findByText('Confirm access grant')).toBeTruthy();
		expect(mockClient.mutation).not.toHaveBeenCalled();
		// No wildcard warning for a scoped grant.
		expect(screen.queryByText(/is a wildcard/)).toBeFalsy();

		fireEvent.click(screen.getByRole('button', { name: 'Confirm grant' }));

		await waitFor(() => expect(mockClient.mutation).toHaveBeenCalledTimes(1));
		expect(mockClient.mutation).toHaveBeenCalledWith(
			expect.anything(),
			expect.objectContaining({
				params: {
					tuples: [
						{ user: 'user:42', relation: 'viewer', object: 'document:9' },
					],
				},
			}),
		);
	});

	it('warns before granting a wildcard (user:*) tuple', async () => {
		mockQueries(emptyTuplesResponse);
		renderTuples();

		await screen.findByText('No relationship tuples yet');

		fireEvent.change(screen.getByLabelText('User'), {
			target: { value: 'user:*' },
		});
		fireEvent.change(screen.getByLabelText('Relation'), {
			target: { value: 'viewer' },
		});
		fireEvent.change(screen.getByLabelText('Object'), {
			target: { value: 'document:9' },
		});
		fireEvent.click(screen.getByRole('button', { name: /^Add$/ }));

		expect(await screen.findByText(/is a wildcard/)).toBeTruthy();
		expect(mockClient.mutation).not.toHaveBeenCalled();
	});

	it('opens a confirm dialog before deleting a tuple', async () => {
		mockQueries(oneTupleResponse);
		renderTuples();

		await screen.findByText('user:1');

		fireEvent.click(screen.getByRole('button', { name: 'Revoke this tuple' }));

		expect(await screen.findByText('Revoke access?')).toBeTruthy();
		expect(mockClient.mutation).not.toHaveBeenCalled();

		fireEvent.click(screen.getByRole('button', { name: 'Revoke' }));

		await waitFor(() => expect(mockClient.mutation).toHaveBeenCalledTimes(1));
	});
});
