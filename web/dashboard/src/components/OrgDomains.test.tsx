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
import OrgDomains from './OrgDomains';
import {
	RequestOrgDomain,
	VerifyOrgDomain,
	DeleteOrgDomain,
} from '../graphql/mutation';

// urql's useClient is the only external dependency OrgDomains.tsx needs — mock
// it so query()/mutation() responses are controlled per-call by the test.
vi.mock('urql', () => ({
	useClient: () => mockClient,
}));

let mockClient: {
	query: ReturnType<typeof vi.fn>;
	mutation: ReturnType<typeof vi.fn>;
};

// The _org_domains list response, mutated in-place so a mutation handler can
// make the post-action refetch observe the new state.
let listResponse: { data: { _org_domains: { org_domains: unknown[] } } };

const challenge = {
	domain: 'acme.com',
	record_type: 'TXT',
	record_name: '_authorizer-challenge.acme.com',
	record_value: 'authorizer-domain-verification=tok123',
};

const verifiedDomain = {
	domain: 'acme.com',
	org_id: 'org1',
	verified_at: 1700000000,
	created_at: 1700000000,
	updated_at: 1700000000,
};

// vitest.config.ts does not set test.globals, so @testing-library/react's
// automatic afterEach(cleanup) never registers — clean up Radix portals by hand.
afterEach(cleanup);

beforeEach(() => {
	listResponse = { data: { _org_domains: { org_domains: [] } } };
	mockClient = {
		query: vi.fn(() => ({
			toPromise: () => Promise.resolve(listResponse),
		})),
		mutation: vi.fn(),
	};
});

describe('OrgDomains', () => {
	it('renders the list of verified domains', async () => {
		listResponse = {
			data: { _org_domains: { org_domains: [verifiedDomain] } },
		};
		render(<OrgDomains orgId="org1" orgSlug="Acme" />);
		expect(await screen.findByText('acme.com')).toBeTruthy();
	});

	it('runs the DNS challenge flow: request → shows TXT → verify → verified', async () => {
		mockClient.mutation.mockImplementation((doc: unknown) => {
			if (doc === RequestOrgDomain) {
				return {
					toPromise: () =>
						Promise.resolve({ data: { _request_org_domain: challenge } }),
				};
			}
			// Verify succeeds; make the subsequent refetch observe the new row.
			listResponse = {
				data: { _org_domains: { org_domains: [verifiedDomain] } },
			};
			return {
				toPromise: () =>
					Promise.resolve({ data: { _verify_org_domain: verifiedDomain } }),
			};
		});

		render(<OrgDomains orgId="org1" orgSlug="Acme" />);
		await screen.findByText('No verified domains yet.');

		fireEvent.click(screen.getByRole('button', { name: /Add Domain/ }));
		fireEvent.change(screen.getByPlaceholderText('acme.com'), {
			target: { value: 'acme.com' },
		});
		fireEvent.click(
			screen.getByRole('button', { name: 'Request DNS Challenge' }),
		);

		// The TXT record to publish is shown.
		expect(
			await screen.findByText('_authorizer-challenge.acme.com'),
		).toBeTruthy();
		expect(
			screen.getByText('authorizer-domain-verification=tok123'),
		).toBeTruthy();

		fireEvent.click(screen.getByRole('button', { name: /Verify/ }));

		// Dialog closes (TXT gone) and the domain now shows in the verified table.
		await waitFor(() =>
			expect(
				screen.queryByText('authorizer-domain-verification=tok123'),
			).toBeNull(),
		);
		expect(await screen.findByText('acme.com')).toBeTruthy();
	});

	it('keeps the challenge dialog open on a retryable "not verified yet" error', async () => {
		mockClient.mutation.mockImplementation((doc: unknown) => {
			if (doc === RequestOrgDomain) {
				return {
					toPromise: () =>
						Promise.resolve({ data: { _request_org_domain: challenge } }),
				};
			}
			// Verify fails because DNS hasn't propagated — resolver leaves the
			// challenge in place.
			return {
				toPromise: () =>
					Promise.resolve({
						error: {
							message:
								'dns verification failed: challenge TXT record not found or does not match',
						},
					}),
			};
		});

		render(<OrgDomains orgId="org1" orgSlug="Acme" />);
		await screen.findByText('No verified domains yet.');

		fireEvent.click(screen.getByRole('button', { name: /Add Domain/ }));
		fireEvent.change(screen.getByPlaceholderText('acme.com'), {
			target: { value: 'acme.com' },
		});
		fireEvent.click(
			screen.getByRole('button', { name: 'Request DNS Challenge' }),
		);
		await screen.findByText('_authorizer-challenge.acme.com');

		fireEvent.click(screen.getByRole('button', { name: /Verify/ }));

		// Retryable hint appears and the TXT record is still on screen.
		expect(
			await screen.findByText(/DNS changes can take a few minutes/),
		).toBeTruthy();
		expect(
			screen.getByText('authorizer-domain-verification=tok123'),
		).toBeTruthy();
		expect(screen.getByRole('button', { name: /Verify/ })).toBeTruthy();
	});

	it('deletes a domain after confirmation', async () => {
		listResponse = {
			data: { _org_domains: { org_domains: [verifiedDomain] } },
		};
		mockClient.mutation.mockImplementation(() => {
			listResponse = { data: { _org_domains: { org_domains: [] } } };
			return {
				toPromise: () =>
					Promise.resolve({ data: { _delete_org_domain: { message: 'ok' } } }),
			};
		});

		render(<OrgDomains orgId="org1" orgSlug="Acme" />);
		await screen.findByText('acme.com');

		// Row delete opens the confirm dialog; the mutation has NOT fired yet.
		fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
		expect(await screen.findByText('Delete Verified Domain')).toBeTruthy();
		expect(mockClient.mutation).not.toHaveBeenCalled();

		// Confirm: two "Delete" buttons now (row + dialog) — click the dialog's.
		const deleteButtons = screen.getAllByRole('button', { name: 'Delete' });
		fireEvent.click(deleteButtons[deleteButtons.length - 1]);

		await waitFor(() =>
			expect(mockClient.mutation).toHaveBeenCalledWith(
				DeleteOrgDomain,
				expect.objectContaining({ params: { domain: 'acme.com' } }),
			),
		);
		await waitFor(() => expect(screen.queryByText('acme.com')).toBeNull());
	});
});
