// @vitest-environment jsdom
import React from 'react';
import { describe, expect, it, vi, afterEach } from 'vitest';
import {
	render,
	screen,
	fireEvent,
	waitFor,
	cleanup,
} from '@testing-library/react';
import Users from './Users';
import { TooltipProvider } from '../components/ui/tooltip';
import type { User } from '../types';

// This project's vitest.config.ts does not set `test.globals: true`, so
// @testing-library/react's automatic afterEach(cleanup) never registers.
afterEach(cleanup);

// Deferred promise so the test controls exactly when each mocked GraphQL
// response resolves, independent of when the request was fired.
function deferred<T>() {
	let resolve!: (value: T) => void;
	const promise = new Promise<T>((res) => {
		resolve = res;
	});
	return { promise, resolve };
}

const makeUser = (id: string, email: string): User => ({
	id,
	email,
	email_verified: true,
	signup_methods: 'basic_auth',
	roles: ['user'],
	created_at: 0,
});

const usersResponse = (users: User[], total: number) => ({
	data: {
		_users: {
			users,
			pagination: { limit: 10, page: 1, offset: 0, total },
		},
	},
});

const adminMetaResponse = {
	data: { _admin_meta: { is_multi_factor_auth_service_enabled: true } },
};

// urql's useClient is the only external dependency Users.tsx needs — mock it
// so query() responses are controlled per-call by the test.
vi.mock('urql', () => ({
	useClient: () => mockClient,
}));

// eslint-disable-next-line prefer-const
let mockClient: { query: ReturnType<typeof vi.fn> };

describe('Users list — stale-response race guard', () => {
	it('keeps the result of the most recently fired request, not the one that resolves last', async () => {
		// Request A: fired on mount, resolves LAST (slow).
		const requestA = deferred<ReturnType<typeof usersResponse>>();
		// Request B: fired by a search change before A resolves, resolves
		// FIRST (fast) — the out-of-order case the guard must handle.
		const requestB = deferred<ReturnType<typeof usersResponse>>();

		let userQueryCall = 0;
		mockClient = {
			query: vi.fn(() => {
				userQueryCall += 1;
				// Call 1 is always AdminRolesQuery (fired once on mount).
				if (userQueryCall === 1) {
					return { toPromise: () => Promise.resolve(adminMetaResponse) };
				}
				// Call 2 is the initial UserDetailsQuery (request A); call 3 is
				// the one triggered by the search change (request B).
				const which = userQueryCall === 2 ? requestA : requestB;
				return { toPromise: () => which.promise };
			}),
		};

		render(
			<TooltipProvider>
				<Users />
			</TooltipProvider>,
		);

		// Request A has fired (mount effect).
		await waitFor(() => expect(mockClient.query).toHaveBeenCalledTimes(2));

		// Trigger request B via the search box, before A resolves. The 300ms
		// debounce means the second query call lands ~300ms after typing.
		fireEvent.change(
			screen.getByPlaceholderText('Search by email, name, or ID...'),
			{ target: { value: 'b' } },
		);
		await waitFor(() => expect(mockClient.query).toHaveBeenCalledTimes(3), {
			timeout: 1000,
		});

		// B (the later-fired request) resolves first.
		requestB.resolve(usersResponse([makeUser('b', 'b@example.com')], 1));
		await waitFor(() =>
			expect(screen.queryByText('b@example.com')).toBeTruthy(),
		);

		// A (the earlier-fired request) resolves after — its data must NOT
		// overwrite the list, since a newer request (B) has since started.
		requestA.resolve(usersResponse([makeUser('a', 'a@example.com')], 1));

		// Give the resolved microtask a tick to flush into state if the guard
		// were absent — confirm it stays absent even after settling.
		await new Promise((r) => setTimeout(r, 50));

		expect(screen.queryByText('a@example.com')).toBeFalsy();
		expect(screen.getByText('b@example.com')).toBeTruthy();
	});
});
