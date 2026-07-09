import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import { Link, useParams } from 'react-router-dom';
import {
	ArrowLeft,
	ChevronLeft,
	ChevronRight,
	Plus,
	RefreshCw,
	Trash2,
} from 'lucide-react';
import dayjs from 'dayjs';
import { toast } from 'sonner';
import UpdateOrgOIDCConnectionModal from '../components/UpdateOrgOIDCConnectionModal';
import UpdateOrgSAMLConnectionModal from '../components/UpdateOrgSAMLConnectionModal';
import ClientSecretDialog from '../components/ClientSecretDialog';
import { UpdateModalViews } from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import {
	OrganizationQuery,
	OrgMembersQuery,
	OrgOIDCConnectionQuery,
	OrgSAMLConnectionQuery,
	ScimEndpointQuery,
} from '../graphql/queries';
import {
	AddOrgMember,
	RemoveOrgMember,
	DeleteOrgOIDCConnection,
	DeleteOrgSAMLConnection,
	CreateScimEndpoint,
	RotateScimToken,
	DeleteScimEndpoint,
} from '../graphql/mutation';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
import { Skeleton } from '../components/ui/skeleton';
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from '../components/ui/card';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from '../components/ui/dialog';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from '../components/ui/table';
import type {
	CreateScimEndpointResponse,
	Organization,
	OrgMember,
	OrgOIDCConnection,
	OrgSAMLConnection,
	ScimEndpoint,
} from '../types';

const MEMBERS_PAGE_LIMIT = 10;

// ponytail: one local confirm dialog instead of a modal file per action.
interface ConfirmDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	title: string;
	body: React.ReactNode;
	actionLabel: string;
	destructive?: boolean;
	onConfirm: () => void;
}

const ConfirmDialog = ({
	open,
	onOpenChange,
	title,
	body,
	actionLabel,
	destructive,
	onConfirm,
}: ConfirmDialogProps) => (
	<Dialog open={open} onOpenChange={onOpenChange}>
		<DialogContent>
			<DialogHeader>
				<DialogTitle>{title}</DialogTitle>
				<DialogDescription>Are you sure?</DialogDescription>
			</DialogHeader>
			<div
				className={
					destructive
						? 'rounded-md border border-red-300 bg-red-50 p-4'
						: 'rounded-md border border-yellow-300 bg-yellow-50 p-4'
				}
			>
				<p className="text-sm">{body}</p>
			</div>
			<DialogFooter>
				<Button
					variant={destructive ? 'destructive' : 'default'}
					onClick={onConfirm}
				>
					{destructive ? (
						<Trash2 className="mr-2 h-4 w-4" />
					) : (
						<RefreshCw className="mr-2 h-4 w-4" />
					)}
					{actionLabel}
				</Button>
			</DialogFooter>
		</DialogContent>
	</Dialog>
);

const parseRoles = (value: string): string[] =>
	value
		.split(/[\s,]+/)
		.map((role) => role.trim())
		.filter(Boolean);

const OrganizationDetail = () => {
	const { id } = useParams<{ id: string }>();
	const client = useClient();

	const [org, setOrg] = useState<Organization | null>(null);
	const [orgLoading, setOrgLoading] = useState(true);

	// Members
	const [members, setMembers] = useState<OrgMember[]>([]);
	const [membersPage, setMembersPage] = useState(1);
	const [membersTotal, setMembersTotal] = useState(0);
	const [newMemberUserId, setNewMemberUserId] = useState('');
	const [newMemberRoles, setNewMemberRoles] = useState('');
	const [addingMember, setAddingMember] = useState(false);
	const [memberToRemove, setMemberToRemove] = useState<OrgMember | null>(null);

	// SSO connections (at most one of each per org)
	const [oidcConnection, setOidcConnection] =
		useState<OrgOIDCConnection | null>(null);
	const [samlConnection, setSamlConnection] =
		useState<OrgSAMLConnection | null>(null);
	const [deleteOidcOpen, setDeleteOidcOpen] = useState(false);
	const [deleteSamlOpen, setDeleteSamlOpen] = useState(false);

	// SCIM
	const [scimEndpoint, setScimEndpoint] = useState<ScimEndpoint | null>(null);
	const [scimToken, setScimToken] = useState<string | null>(null);
	const [rotateScimOpen, setRotateScimOpen] = useState(false);
	const [deleteScimOpen, setDeleteScimOpen] = useState(false);
	const [scimLoading, setScimLoading] = useState(false);

	const orgId = org?.id || '';

	const fetchOrg = async () => {
		if (!id) return;
		setOrgLoading(true);
		const res = await client
			.query<{ _organization: Organization }>(OrganizationQuery, {
				params: { id },
			})
			.toPromise();
		setOrg(res.data?._organization || null);
		setOrgLoading(false);
	};

	const fetchMembers = async (page = membersPage) => {
		if (!id) return;
		const res = await client
			.query<{
				_org_members: {
					org_members: OrgMember[];
					pagination: { total: number };
				};
			}>(OrgMembersQuery, {
				params: {
					org_id: id,
					pagination: {
						pagination: { limit: MEMBERS_PAGE_LIMIT, page },
					},
				},
			})
			.toPromise();
		const list = res.data?._org_members?.org_members || [];
		setMembers(list);
		setMembersTotal(res.data?._org_members?.pagination?.total || 0);
		// removing the last member of a page > 1 leaves it out of range
		if (list.length === 0 && page > 1) {
			setMembersPage(1);
		}
	};

	// Fetching a missing connection/endpoint errors on the server; treat any
	// error as "not configured".
	const fetchOidcConnection = async () => {
		if (!id) return;
		const res = await client
			.query<{
				_org_oidc_connection: OrgOIDCConnection;
			}>(OrgOIDCConnectionQuery, { params: { org_id: id } })
			.toPromise();
		setOidcConnection(res.data?._org_oidc_connection || null);
	};

	const fetchSamlConnection = async () => {
		if (!id) return;
		const res = await client
			.query<{
				_org_saml_connection: OrgSAMLConnection;
			}>(OrgSAMLConnectionQuery, { params: { org_id: id } })
			.toPromise();
		setSamlConnection(res.data?._org_saml_connection || null);
	};

	const fetchScimEndpoint = async () => {
		if (!id) return;
		const res = await client
			.query<{ _scim_endpoint: ScimEndpoint }>(ScimEndpointQuery, {
				params: { org_id: id },
			})
			.toPromise();
		setScimEndpoint(res.data?._scim_endpoint || null);
	};

	useEffect(() => {
		fetchOrg();
		fetchOidcConnection();
		fetchSamlConnection();
		fetchScimEndpoint();
	}, [id]);

	useEffect(() => {
		fetchMembers();
	}, [id, membersPage]);

	const maxMembersPages = Math.max(
		1,
		Math.ceil(membersTotal / MEMBERS_PAGE_LIMIT),
	);

	const addMemberHandler = async () => {
		if (!newMemberUserId.trim()) return;
		setAddingMember(true);
		const res = await client
			.mutation(AddOrgMember, {
				params: {
					org_id: id,
					user_id: newMemberUserId.trim(),
					roles: parseRoles(newMemberRoles),
				},
			})
			.toPromise();
		setAddingMember(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to add member'),
				),
			);
			return;
		}
		toast.success('Member added');
		setNewMemberUserId('');
		setNewMemberRoles('');
		fetchMembers();
	};

	const removeMemberHandler = async () => {
		if (!memberToRemove) return;
		const res = await client
			.mutation(RemoveOrgMember, {
				params: { org_id: id, user_id: memberToRemove.user_id },
			})
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to remove member'),
				),
			);
			return;
		}
		toast.success('Member removed');
		setMemberToRemove(null);
		fetchMembers();
	};

	const deleteOidcHandler = async () => {
		const res = await client
			.mutation(DeleteOrgOIDCConnection, { params: { org_id: id } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to delete OIDC connection'),
				),
			);
			return;
		}
		toast.success('OIDC connection deleted');
		setDeleteOidcOpen(false);
		setOidcConnection(null);
	};

	const deleteSamlHandler = async () => {
		const res = await client
			.mutation(DeleteOrgSAMLConnection, { params: { org_id: id } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to delete SAML connection'),
				),
			);
			return;
		}
		toast.success('SAML connection deleted');
		setDeleteSamlOpen(false);
		setSamlConnection(null);
	};

	const createScimHandler = async () => {
		setScimLoading(true);
		const res = await client
			.mutation<{
				_create_scim_endpoint: CreateScimEndpointResponse;
			}>(CreateScimEndpoint, { params: { org_id: id } })
			.toPromise();
		setScimLoading(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to create SCIM endpoint'),
				),
			);
			return;
		}
		toast.success('SCIM endpoint created');
		setScimToken(res.data?._create_scim_endpoint?.token || null);
		fetchScimEndpoint();
	};

	const rotateScimHandler = async () => {
		const res = await client
			.mutation<{
				_rotate_scim_token: CreateScimEndpointResponse;
			}>(RotateScimToken, { params: { org_id: id } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to rotate SCIM token'),
				),
			);
			return;
		}
		toast.success('SCIM token rotated');
		setRotateScimOpen(false);
		setScimToken(res.data?._rotate_scim_token?.token || null);
	};

	const deleteScimHandler = async () => {
		const res = await client
			.mutation(DeleteScimEndpoint, { params: { org_id: id } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to delete SCIM endpoint'),
				),
			);
			return;
		}
		toast.success('SCIM endpoint deleted');
		setDeleteScimOpen(false);
		setScimEndpoint(null);
	};

	if (orgLoading) {
		return (
			<div className="m-5 rounded-md bg-white py-5 px-10 min-h-[25vh] space-y-3">
				{[1, 2, 3].map((i) => (
					<Skeleton key={i} className="h-10 w-full" />
				))}
			</div>
		);
	}

	if (!org) {
		return (
			<div className="m-5 rounded-md bg-white py-5 px-10">
				<p className="text-sm text-gray-500">Organization not found.</p>
				<Link
					to="/identity/organizations"
					className="mt-2 inline-flex items-center text-sm text-blue-600 hover:underline"
				>
					<ArrowLeft className="mr-1 h-4 w-4" />
					Back to Organizations
				</Link>
			</div>
		);
	}

	return (
		<div className="m-5 space-y-5">
			<div className="rounded-md bg-white py-5 px-10">
				<Link
					to="/identity/organizations"
					className="inline-flex items-center text-sm text-blue-600 hover:underline"
				>
					<ArrowLeft className="mr-1 h-4 w-4" />
					Back to Organizations
				</Link>
				<div className="mt-2 flex items-center gap-3">
					<h1 className="text-2xl font-semibold text-gray-900">
						{org.display_name || org.name}
					</h1>
					<Badge variant={org.enabled ? 'success' : 'warning'}>
						{org.enabled ? 'enabled' : 'disabled'}
					</Badge>
				</div>
				<p className="mt-1 text-sm text-gray-500">
					<span className="font-mono text-xs">{org.name}</span>
					{org.created_at
						? ` · created ${dayjs.unix(org.created_at).format('MMM D, YYYY')}`
						: ''}
				</p>
			</div>

			{/* Members */}
			<Card>
				<CardHeader>
					<CardTitle>Members</CardTitle>
					<CardDescription>
						Users belonging to this organization with their per-org roles.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<div className="mb-4 flex items-end gap-3">
						<div className="flex-1">
							<label className="text-sm font-medium">User ID</label>
							<Input
								placeholder="ID of an existing user"
								value={newMemberUserId}
								onChange={(e) => setNewMemberUserId(e.currentTarget.value)}
							/>
						</div>
						<div className="flex-1">
							<label className="text-sm font-medium">Roles</label>
							<Input
								placeholder="admin member (space or comma separated)"
								value={newMemberRoles}
								onChange={(e) => setNewMemberRoles(e.currentTarget.value)}
							/>
						</div>
						<Button
							size="sm"
							onClick={addMemberHandler}
							isLoading={addingMember}
							disabled={addingMember || !newMemberUserId.trim()}
						>
							<Plus className="mr-2 h-4 w-4" />
							Add Member
						</Button>
					</div>
					{members.length ? (
						<>
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>User ID</TableHead>
										<TableHead>Roles</TableHead>
										<TableHead>Added</TableHead>
										<TableHead>Actions</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{members.map((member) => (
										<TableRow key={member.id}>
											<TableCell className="max-w-[300px] text-sm">
												<span className="truncate font-mono text-xs">
													{member.user_id}
												</span>
											</TableCell>
											<TableCell className="max-w-[300px]">
												<div className="flex flex-wrap gap-1">
													{member.roles.length ? (
														member.roles.map((role) => (
															<Badge key={role} variant="secondary">
																{role}
															</Badge>
														))
													) : (
														<span className="text-sm text-gray-400">—</span>
													)}
												</div>
											</TableCell>
											<TableCell className="text-sm whitespace-nowrap">
												{member.created_at
													? dayjs.unix(member.created_at).format('MMM D, YYYY')
													: '—'}
											</TableCell>
											<TableCell>
												<Button
													variant="ghost"
													size="sm"
													onClick={() => setMemberToRemove(member)}
												>
													Remove
												</Button>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
							{maxMembersPages > 1 && (
								<div className="mt-4 flex items-center justify-end gap-3 text-sm">
									<span>
										Page <strong>{membersPage}</strong> of{' '}
										<strong>{maxMembersPages}</strong>
									</span>
									<div className="flex gap-1">
										<Button
											variant="outline"
											size="icon"
											onClick={() => setMembersPage(membersPage - 1)}
											disabled={membersPage <= 1}
										>
											<ChevronLeft className="h-4 w-4" />
										</Button>
										<Button
											variant="outline"
											size="icon"
											onClick={() => setMembersPage(membersPage + 1)}
											disabled={membersPage >= maxMembersPages}
										>
											<ChevronRight className="h-4 w-4" />
										</Button>
									</div>
								</div>
							)}
						</>
					) : (
						<p className="text-sm text-gray-400">No members yet.</p>
					)}
				</CardContent>
			</Card>

			{/* SSO: OIDC */}
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>SSO — OIDC Connection</CardTitle>
							<CardDescription>
								Upstream OIDC identity provider brokered for this organization
								(one per org).
							</CardDescription>
						</div>
						{oidcConnection ? (
							<div className="flex gap-2">
								<UpdateOrgOIDCConnectionModal
									view={UpdateModalViews.Edit}
									orgId={orgId}
									selectedConnection={oidcConnection}
									fetchConnection={fetchOidcConnection}
								/>
								<Button
									variant="destructive"
									size="sm"
									onClick={() => setDeleteOidcOpen(true)}
								>
									Delete
								</Button>
							</div>
						) : (
							<UpdateOrgOIDCConnectionModal
								view={UpdateModalViews.ADD}
								orgId={orgId}
								fetchConnection={fetchOidcConnection}
							/>
						)}
					</div>
				</CardHeader>
				<CardContent>
					{oidcConnection ? (
						<dl className="grid grid-cols-1 gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
							<div>
								<dt className="font-medium text-gray-500">Name</dt>
								<dd>{oidcConnection.name}</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Active</dt>
								<dd>
									<Badge
										variant={oidcConnection.is_active ? 'success' : 'warning'}
									>
										{oidcConnection.is_active.toString()}
									</Badge>
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Issuer URL</dt>
								<dd className="font-mono text-xs">
									{oidcConnection.issuer_url}
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Client ID</dt>
								<dd className="font-mono text-xs">
									{oidcConnection.sso_client_id}
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Scopes</dt>
								<dd>{oidcConnection.scopes || 'openid profile email'}</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Redirect URI</dt>
								<dd className="font-mono text-xs">
									{oidcConnection.redirect_uri || 'Derived from request host'}
								</dd>
							</div>
						</dl>
					) : (
						<p className="text-sm text-gray-400">
							No OIDC connection configured.
						</p>
					)}
				</CardContent>
			</Card>

			{/* SSO: SAML */}
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>SSO — SAML Connection</CardTitle>
							<CardDescription>
								Upstream SAML 2.0 identity provider for this organization (one
								per org). Authorizer acts as the Service Provider.
							</CardDescription>
						</div>
						{samlConnection ? (
							<div className="flex gap-2">
								<UpdateOrgSAMLConnectionModal
									view={UpdateModalViews.Edit}
									orgId={orgId}
									selectedConnection={samlConnection}
									fetchConnection={fetchSamlConnection}
								/>
								<Button
									variant="destructive"
									size="sm"
									onClick={() => setDeleteSamlOpen(true)}
								>
									Delete
								</Button>
							</div>
						) : (
							<UpdateOrgSAMLConnectionModal
								view={UpdateModalViews.ADD}
								orgId={orgId}
								fetchConnection={fetchSamlConnection}
							/>
						)}
					</div>
				</CardHeader>
				<CardContent>
					{samlConnection ? (
						<dl className="grid grid-cols-1 gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
							<div>
								<dt className="font-medium text-gray-500">Name</dt>
								<dd>{samlConnection.name}</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Active</dt>
								<dd>
									<Badge
										variant={samlConnection.is_active ? 'success' : 'warning'}
									>
										{samlConnection.is_active.toString()}
									</Badge>
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">IdP Entity ID</dt>
								<dd className="font-mono text-xs">
									{samlConnection.idp_entity_id}
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">IdP SSO URL</dt>
								<dd className="font-mono text-xs">
									{samlConnection.idp_sso_url || '—'}
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">SP Entity ID</dt>
								<dd className="font-mono text-xs">
									{samlConnection.sp_entity_id || 'Derived from request host'}
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">ACS URL</dt>
								<dd className="font-mono text-xs">
									{samlConnection.acs_url || 'Derived from request host'}
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">IdP Initiated</dt>
								<dd>
									{samlConnection.allow_idp_initiated ? 'Allowed' : 'Disabled'}
								</dd>
							</div>
						</dl>
					) : (
						<p className="text-sm text-gray-400">
							No SAML connection configured.
						</p>
					)}
				</CardContent>
			</Card>

			{/* SCIM */}
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>SCIM Provisioning</CardTitle>
							<CardDescription>
								Inbound SCIM 2.0 endpoint credential for this organization (one
								per org). The bearer token is shown only once.
							</CardDescription>
						</div>
						{scimEndpoint ? (
							<div className="flex gap-2">
								<Button
									variant="outline"
									size="sm"
									onClick={() => setRotateScimOpen(true)}
								>
									<RefreshCw className="mr-2 h-4 w-4" />
									Rotate Token
								</Button>
								<Button
									variant="destructive"
									size="sm"
									onClick={() => setDeleteScimOpen(true)}
								>
									Delete
								</Button>
							</div>
						) : (
							<Button
								size="sm"
								onClick={createScimHandler}
								isLoading={scimLoading}
							>
								<Plus className="mr-2 h-4 w-4" />
								Create SCIM Endpoint
							</Button>
						)}
					</div>
				</CardHeader>
				<CardContent>
					{scimEndpoint ? (
						<dl className="grid grid-cols-1 gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
							<div>
								<dt className="font-medium text-gray-500">Endpoint ID</dt>
								<dd className="font-mono text-xs">{scimEndpoint.id}</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Enabled</dt>
								<dd>
									<Badge variant={scimEndpoint.enabled ? 'success' : 'warning'}>
										{scimEndpoint.enabled.toString()}
									</Badge>
								</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Base URL</dt>
								<dd className="font-mono text-xs">/scim/v2/</dd>
							</div>
							<div>
								<dt className="font-medium text-gray-500">Created</dt>
								<dd>
									{scimEndpoint.created_at
										? dayjs.unix(scimEndpoint.created_at).format('MMM D, YYYY')
										: '—'}
								</dd>
							</div>
						</dl>
					) : (
						<p className="text-sm text-gray-400">
							No SCIM endpoint configured.
						</p>
					)}
				</CardContent>
			</Card>

			{/* Confirm dialogs */}
			<ConfirmDialog
				open={!!memberToRemove}
				onOpenChange={(isOpen) => {
					if (!isOpen) setMemberToRemove(null);
				}}
				title="Remove Member"
				body={
					<>
						User <strong>{memberToRemove?.user_id}</strong> will be removed from
						this organization.
					</>
				}
				actionLabel="Remove"
				destructive
				onConfirm={removeMemberHandler}
			/>
			<ConfirmDialog
				open={deleteOidcOpen}
				onOpenChange={setDeleteOidcOpen}
				title="Delete OIDC Connection"
				body={
					<>
						OIDC connection <strong>{oidcConnection?.name}</strong> will be
						deleted permanently! SSO logins through it will stop working.
					</>
				}
				actionLabel="Delete"
				destructive
				onConfirm={deleteOidcHandler}
			/>
			<ConfirmDialog
				open={deleteSamlOpen}
				onOpenChange={setDeleteSamlOpen}
				title="Delete SAML Connection"
				body={
					<>
						SAML connection <strong>{samlConnection?.name}</strong> will be
						deleted permanently! SSO logins through it will stop working.
					</>
				}
				actionLabel="Delete"
				destructive
				onConfirm={deleteSamlHandler}
			/>
			<ConfirmDialog
				open={rotateScimOpen}
				onOpenChange={setRotateScimOpen}
				title="Rotate SCIM Token"
				body="The current SCIM token will stop working immediately. The new token is shown only once."
				actionLabel="Rotate Token"
				onConfirm={rotateScimHandler}
			/>
			<ConfirmDialog
				open={deleteScimOpen}
				onOpenChange={setDeleteScimOpen}
				title="Delete SCIM Endpoint"
				body="The SCIM endpoint and its token will be deleted permanently! Provisioning from the org's IdP will stop working."
				actionLabel="Delete"
				destructive
				onConfirm={deleteScimHandler}
			/>

			{/* One-time SCIM token display */}
			<ClientSecretDialog
				secret={scimToken}
				onClose={() => setScimToken(null)}
				label="SCIM Token"
			/>
		</div>
	);
};

export default OrganizationDetail;
