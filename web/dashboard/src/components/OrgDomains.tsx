import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { Plus, Copy, Check, ShieldCheck } from 'lucide-react';
import dayjs from 'dayjs';
import {
	capitalizeFirstLetter,
	copyTextToClipboard,
	getGraphQLErrorMessage,
} from '../utils';
import { OrgDomainsQuery } from '../graphql/queries';
import {
	RequestOrgDomain,
	VerifyOrgDomain,
	AddVerifiedOrgDomain,
	DeleteOrgDomain,
} from '../graphql/mutation';
import type { OrgDomain, OrgDomainChallenge } from '../types';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Input } from './ui/input';
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from './ui/card';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from './ui/dialog';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from './ui/table';

interface OrgDomainsProps {
	orgId: string;
	// orgSlug is the display name of the org, shown in copy for context.
	orgSlug: string;
}

// CopyField renders a read-only value with a copy-to-clipboard affordance,
// mirroring the inline copy control in ClientSecretDialog.
const CopyField = ({ label, value }: { label: string; value: string }) => {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		await copyTextToClipboard(value);
		setCopied(true);
		toast.success(`${label} copied`);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<div>
			<p className="text-xs font-medium text-gray-500">{label}</p>
			<div className="mt-1 flex items-center gap-2 rounded-md bg-gray-100 p-3">
				<code className="flex-1 break-all font-mono text-sm">{value}</code>
				<button
					type="button"
					onClick={handleCopy}
					className="text-gray-400 hover:text-gray-600"
					aria-label={`Copy ${label.toLowerCase()}`}
				>
					{copied ? (
						<Check className="h-4 w-4 text-green-500" />
					) : (
						<Copy className="h-4 w-4" />
					)}
				</button>
			</div>
		</div>
	);
};

const OrgDomains = ({ orgId, orgSlug }: OrgDomainsProps) => {
	const client = useClient();

	const [domains, setDomains] = useState<OrgDomain[]>([]);
	const [domainsLoading, setDomainsLoading] = useState(true);

	// Add-domain dialog. `challenge` null => domain-entry step; set => DNS step.
	const [addOpen, setAddOpen] = useState(false);
	const [domainInput, setDomainInput] = useState('');
	const [challenge, setChallenge] = useState<OrgDomainChallenge | null>(null);
	const [requesting, setRequesting] = useState(false);
	const [verifying, setVerifying] = useState(false);
	const [addingVerified, setAddingVerified] = useState(false);
	// Retryable "DNS not propagated yet" hint shown inline without discarding
	// the challenge so the tenant can retry after the record propagates.
	const [notVerifiedHint, setNotVerifiedHint] = useState(false);

	const [domainToDelete, setDomainToDelete] = useState<OrgDomain | null>(null);

	const fetchDomains = async () => {
		if (!orgId) return;
		setDomainsLoading(true);
		const res = await client
			.query<{ _org_domains: { org_domains: OrgDomain[] } }>(OrgDomainsQuery, {
				// Verified domains per org are expected to be a handful at most; a
				// generous single-page limit avoids silently truncating at the
				// backend's default of 10 without needing full pagination UI.
				params: { org_id: orgId, pagination: { limit: 100 } },
			})
			.toPromise();
		setDomains(res.data?._org_domains?.org_domains || []);
		setDomainsLoading(false);
	};

	useEffect(() => {
		fetchDomains();
	}, [orgId]);

	const resetAddDialog = () => {
		setDomainInput('');
		setChallenge(null);
		setNotVerifiedHint(false);
	};

	const requestChallengeHandler = async () => {
		if (!domainInput.trim()) return;
		setRequesting(true);
		const res = await client
			.mutation<{ _request_org_domain: OrgDomainChallenge }>(RequestOrgDomain, {
				params: { org_id: orgId, domain: domainInput.trim() },
			})
			.toPromise();
		setRequesting(false);
		if (res.error || !res.data?._request_org_domain) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(
						res.error,
						'Failed to request domain challenge',
					),
				),
			);
			return;
		}
		setChallenge(res.data._request_org_domain);
		setNotVerifiedHint(false);
	};

	const verifyHandler = async () => {
		if (!challenge) return;
		setVerifying(true);
		const res = await client
			.mutation<{ _verify_org_domain: OrgDomain }>(VerifyOrgDomain, {
				params: { org_id: orgId, domain: challenge.domain },
			})
			.toPromise();
		setVerifying(false);
		if (res.error) {
			const msg = getGraphQLErrorMessage(res.error, 'Verification failed');
			// The resolver leaves the challenge in place when the TXT record isn't
			// resolvable yet ("dns verification failed: ..."). Surface it as a
			// retryable hint and keep the dialog + record on screen.
			if (/dns verification failed/i.test(msg)) {
				setNotVerifiedHint(true);
			} else {
				toast.error(capitalizeFirstLetter(msg));
			}
			return;
		}
		toast.success('Domain verified');
		setAddOpen(false);
		resetAddDialog();
		fetchDomains();
	};

	const quickAddHandler = async () => {
		if (!domainInput.trim()) return;
		setAddingVerified(true);
		const res = await client
			.mutation<{ _add_verified_org_domain: OrgDomain }>(AddVerifiedOrgDomain, {
				params: { org_id: orgId, domain: domainInput.trim() },
			})
			.toPromise();
		setAddingVerified(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to add verified domain'),
				),
			);
			return;
		}
		toast.success('Domain added');
		setAddOpen(false);
		resetAddDialog();
		fetchDomains();
	};

	const deleteHandler = async () => {
		if (!domainToDelete) return;
		const res = await client
			.mutation(DeleteOrgDomain, {
				params: { domain: domainToDelete.domain },
			})
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to delete domain'),
				),
			);
			return;
		}
		toast.success('Domain deleted');
		setDomainToDelete(null);
		fetchDomains();
	};

	return (
		<>
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>Verified Domains</CardTitle>
							<CardDescription>
								Email domains verified for {orgSlug || 'this organization'}. A
								login from a verified domain is routed to this org's SSO
								(home-realm discovery).
							</CardDescription>
						</div>
						<Button
							size="sm"
							onClick={() => {
								resetAddDialog();
								setAddOpen(true);
							}}
						>
							<Plus className="mr-2 h-4 w-4" />
							Add Domain
						</Button>
					</div>
				</CardHeader>
				<CardContent>
					{domainsLoading ? (
						<p className="text-sm text-gray-400">Loading verified domains…</p>
					) : domains.length > 0 ? (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Domain</TableHead>
									<TableHead>Verified</TableHead>
									<TableHead className="text-right">Actions</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{domains.map((d) => (
									<TableRow key={d.domain}>
										<TableCell className="font-mono text-xs">
											{d.domain}
										</TableCell>
										<TableCell>
											{d.verified_at ? (
												<Badge variant="success">
													{dayjs.unix(d.verified_at).format('MMM D, YYYY')}
												</Badge>
											) : (
												<span className="text-sm text-gray-400">—</span>
											)}
										</TableCell>
										<TableCell className="text-right">
											<Button
												variant="destructive"
												size="sm"
												onClick={() => setDomainToDelete(d)}
											>
												Delete
											</Button>
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					) : (
						<p className="text-sm text-gray-400">No verified domains yet.</p>
					)}
				</CardContent>
			</Card>

			{/* Add domain: DNS challenge flow (primary) + quick verify (super-admin) */}
			<Dialog
				open={addOpen}
				onOpenChange={(isOpen) => {
					setAddOpen(isOpen);
					if (!isOpen) resetAddDialog();
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Add Verified Domain</DialogTitle>
						<DialogDescription>
							{challenge
								? 'Publish this DNS TXT record, then verify.'
								: 'Prove control of a domain by publishing a DNS TXT record.'}
						</DialogDescription>
					</DialogHeader>

					{challenge ? (
						<div className="space-y-4">
							<div className="rounded-md border border-blue-200 bg-blue-50 p-4">
								<p className="text-sm text-blue-800">
									Create the following <strong>{challenge.record_type}</strong>{' '}
									record for <strong>{challenge.domain}</strong>, then click
									Verify.
								</p>
							</div>
							<CopyField label="Record name" value={challenge.record_name} />
							<CopyField label="Record value" value={challenge.record_value} />
							{notVerifiedHint && (
								<div className="rounded-md border border-yellow-300 bg-yellow-50 p-4">
									<p className="text-sm text-yellow-800">
										Not verified yet. DNS changes can take a few minutes to
										propagate — leave the record in place and try again shortly.
									</p>
								</div>
							)}
						</div>
					) : (
						<div className="space-y-3">
							<div>
								<label
									htmlFor="org-domain-input"
									className="text-sm font-medium"
								>
									Domain
								</label>
								<Input
									id="org-domain-input"
									placeholder="acme.com"
									value={domainInput}
									onChange={(e) => setDomainInput(e.currentTarget.value)}
								/>
							</div>
							<button
								type="button"
								onClick={quickAddHandler}
								disabled={addingVerified || !domainInput.trim()}
								className="inline-flex items-center text-sm text-blue-600 hover:underline disabled:cursor-not-allowed disabled:opacity-50"
							>
								<ShieldCheck className="mr-1 h-4 w-4" />
								Add without DNS verification (super-admin)
							</button>
						</div>
					)}

					<DialogFooter>
						{challenge ? (
							<Button
								onClick={verifyHandler}
								isLoading={verifying}
								disabled={verifying}
							>
								I've published this record — Verify
							</Button>
						) : (
							<Button
								onClick={requestChallengeHandler}
								isLoading={requesting}
								disabled={requesting || !domainInput.trim()}
							>
								Request DNS Challenge
							</Button>
						)}
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Confirm: delete domain */}
			<Dialog
				open={!!domainToDelete}
				onOpenChange={(isOpen) => {
					if (!isOpen) setDomainToDelete(null);
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete Verified Domain</DialogTitle>
						<DialogDescription>Are you sure?</DialogDescription>
					</DialogHeader>
					<div className="rounded-md border border-red-300 bg-red-50 p-4">
						<p className="text-sm">
							Domain <strong>{domainToDelete?.domain}</strong> will be removed.
							Logins from it will no longer route to this organization's SSO.
						</p>
					</div>
					<DialogFooter>
						<Button variant="destructive" onClick={deleteHandler}>
							Delete
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
};

export default OrgDomains;
