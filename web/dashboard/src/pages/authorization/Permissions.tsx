import React from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import { Plus, Pencil, Trash2, AlertCircle } from 'lucide-react';
import {
	PermissionsQuery,
	ResourcesQuery,
	ScopesQuery,
	PoliciesQuery,
} from '../../graphql/queries';
import {
	AddPermission,
	UpdatePermission,
	DeletePermission,
} from '../../graphql/mutation';
import { getGraphQLErrorMessage } from '../../utils';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Textarea } from '../../components/ui/textarea';
import { Label } from '../../components/ui/label';
import { Badge } from '../../components/ui/badge';
import { Skeleton } from '../../components/ui/skeleton';
import { Select } from '../../components/ui/select';
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogFooter,
	DialogDescription,
} from '../../components/ui/dialog';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from '../../components/ui/table';
import type {
	AuthzPermission,
	AuthzPermissionsResponse,
	AuthzResource,
	AuthzResourcesResponse,
	AuthzScope,
	AuthzScopesResponse,
	AuthzPolicy,
	AuthzPoliciesResponse,
} from '../../types';

export default function Permissions() {
	const client = useClient();
	const [permissions, setPermissions] = React.useState<AuthzPermission[]>([]);
	const [allResources, setAllResources] = React.useState<AuthzResource[]>([]);
	const [allScopes, setAllScopes] = React.useState<AuthzScope[]>([]);
	const [allPolicies, setAllPolicies] = React.useState<AuthzPolicy[]>([]);
	const [loading, setLoading] = React.useState(false);
	const [dialogOpen, setDialogOpen] = React.useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
	const [editingPermission, setEditingPermission] =
		React.useState<AuthzPermission | null>(null);
	const [deletingPermission, setDeletingPermission] =
		React.useState<AuthzPermission | null>(null);
	const [formName, setFormName] = React.useState('');
	const [formDescription, setFormDescription] = React.useState('');
	const [formResourceId, setFormResourceId] = React.useState('');
	const [formScopeIds, setFormScopeIds] = React.useState<string[]>([]);
	const [formPolicyIds, setFormPolicyIds] = React.useState<string[]>([]);
	const [formDecisionStrategy, setFormDecisionStrategy] =
		React.useState('affirmative');
	const [saving, setSaving] = React.useState(false);

	const fetchPermissions = async () => {
		setLoading(true);
		const { data, error } = await client
			.query<AuthzPermissionsResponse>(PermissionsQuery, {
				params: { pagination: { limit: 100, page: 1 } },
			})
			.toPromise();
		if (data?._permissions) {
			setPermissions(data._permissions.permissions);
		}
		if (error) {
			toast.error(
				getGraphQLErrorMessage(error, 'Failed to load permissions'),
			);
		}
		setLoading(false);
	};

	const fetchRelatedData = async () => {
		const [resourcesRes, scopesRes, policiesRes] = await Promise.all([
			client
				.query<AuthzResourcesResponse>(ResourcesQuery, {
					params: { pagination: { limit: 100, page: 1 } },
				})
				.toPromise(),
			client
				.query<AuthzScopesResponse>(ScopesQuery, {
					params: { pagination: { limit: 100, page: 1 } },
				})
				.toPromise(),
			client
				.query<AuthzPoliciesResponse>(PoliciesQuery, {
					params: { pagination: { limit: 100, page: 1 } },
				})
				.toPromise(),
		]);
		if (resourcesRes.data?._resources) {
			setAllResources(resourcesRes.data._resources.resources);
		}
		if (scopesRes.data?._scopes) {
			setAllScopes(scopesRes.data._scopes.scopes);
		}
		if (policiesRes.data?._policies) {
			setAllPolicies(policiesRes.data._policies.policies);
		}
	};

	React.useEffect(() => {
		fetchPermissions();
		fetchRelatedData();
	}, []);

	const openAddDialog = () => {
		setEditingPermission(null);
		setFormName('');
		setFormDescription('');
		setFormResourceId('');
		setFormScopeIds([]);
		setFormPolicyIds([]);
		setFormDecisionStrategy('affirmative');
		setDialogOpen(true);
	};

	const openEditDialog = (permission: AuthzPermission) => {
		setEditingPermission(permission);
		setFormName(permission.name);
		setFormDescription(permission.description || '');
		setFormResourceId(permission.resource.id);
		setFormScopeIds(permission.scopes.map((s) => s.id));
		setFormPolicyIds(permission.policies.map((p) => p.id));
		setFormDecisionStrategy(permission.decision_strategy);
		setDialogOpen(true);
	};

	const openDeleteDialog = (permission: AuthzPermission) => {
		setDeletingPermission(permission);
		setDeleteDialogOpen(true);
	};

	const toggleScopeId = (id: string) => {
		setFormScopeIds((prev) =>
			prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id],
		);
	};

	const togglePolicyId = (id: string) => {
		setFormPolicyIds((prev) =>
			prev.includes(id) ? prev.filter((p) => p !== id) : [...prev, id],
		);
	};

	const handleSave = async () => {
		if (!formName.trim()) {
			toast.error('Name is required');
			return;
		}
		if (!formResourceId) {
			toast.error('Resource is required');
			return;
		}
		if (formScopeIds.length === 0) {
			toast.error('At least one scope is required');
			return;
		}
		if (formPolicyIds.length === 0) {
			toast.error('At least one policy is required');
			return;
		}
		setSaving(true);
		if (editingPermission) {
			const { error } = await client
				.mutation(UpdatePermission, {
					params: {
						id: editingPermission.id,
						name: formName,
						description: formDescription || undefined,
						scope_ids: formScopeIds,
						policy_ids: formPolicyIds,
						decision_strategy: formDecisionStrategy,
					},
				})
				.toPromise();
			if (error) {
				toast.error(
					getGraphQLErrorMessage(error, 'Failed to update permission'),
				);
			} else {
				toast.success('Permission updated');
				setDialogOpen(false);
				fetchPermissions();
			}
		} else {
			const { error } = await client
				.mutation(AddPermission, {
					params: {
						name: formName,
						description: formDescription || undefined,
						resource_id: formResourceId,
						scope_ids: formScopeIds,
						policy_ids: formPolicyIds,
						decision_strategy: formDecisionStrategy,
					},
				})
				.toPromise();
			if (error) {
				toast.error(
					getGraphQLErrorMessage(error, 'Failed to add permission'),
				);
			} else {
				toast.success('Permission added');
				setDialogOpen(false);
				fetchPermissions();
			}
		}
		setSaving(false);
	};

	const handleDelete = async () => {
		if (!deletingPermission) return;
		setSaving(true);
		const { error } = await client
			.mutation(DeletePermission, { id: deletingPermission.id })
			.toPromise();
		if (error) {
			toast.error(
				getGraphQLErrorMessage(error, 'Failed to delete permission'),
			);
		} else {
			toast.success('Permission deleted');
			setDeleteDialogOpen(false);
			setDeletingPermission(null);
			fetchPermissions();
		}
		setSaving(false);
	};

	// Build natural language summary
	const buildSummary = (): string => {
		const resource = allResources.find((r) => r.id === formResourceId);
		const scopes = allScopes.filter((s) => formScopeIds.includes(s.id));
		const policies = allPolicies.filter((p) => formPolicyIds.includes(p.id));
		if (!resource || scopes.length === 0 || policies.length === 0) {
			return 'Select a resource, scopes, and policies to see a summary.';
		}
		const scopeNames = scopes.map((s) => s.name).join(', ');
		const policyNames = policies.map((p) => `${p.name} (${p.type})`).join(', ');
		const strategy =
			formDecisionStrategy === 'affirmative'
				? 'any matching policy grants access'
				: 'all policies must match';
		return `Allow ${scopeNames} on "${resource.name}" when ${policyNames} — ${strategy}.`;
	};

	return (
		<div>
			<div className="flex items-center justify-between mb-4">
				<div>
					<h2 className="text-lg font-semibold text-gray-900">Permissions</h2>
					<p className="text-sm text-gray-500">
						Combine resources, scopes, and policies into permission rules.
					</p>
				</div>
				<Button onClick={openAddDialog}>
					<Plus className="h-4 w-4 mr-2" />
					Add Permission
				</Button>
			</div>

			{loading ? (
				<div className="space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : permissions.length > 0 ? (
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Name</TableHead>
							<TableHead>Resource</TableHead>
							<TableHead>Scopes</TableHead>
							<TableHead>Policies</TableHead>
							<TableHead>Strategy</TableHead>
							<TableHead>Created</TableHead>
							<TableHead className="w-[100px]">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{permissions.map((perm) => (
							<TableRow key={perm.id}>
								<TableCell className="font-medium">{perm.name}</TableCell>
								<TableCell>
									<Badge>{perm.resource.name}</Badge>
								</TableCell>
								<TableCell>
									<div className="flex flex-wrap gap-1">
										{perm.scopes.map((s) => (
											<Badge key={s.id} variant="secondary">
												{s.name}
											</Badge>
										))}
									</div>
								</TableCell>
								<TableCell>
									<div className="flex flex-wrap gap-1">
										{perm.policies.map((p) => (
											<Badge
												key={p.id}
												variant={
													p.type === 'role' ? 'default' : 'outline'
												}
											>
												{p.name}
											</Badge>
										))}
									</div>
								</TableCell>
								<TableCell className="text-sm">
									{perm.decision_strategy === 'affirmative'
										? 'Any match'
										: 'All must match'}
								</TableCell>
								<TableCell className="text-sm">
									{dayjs(perm.created_at * 1000).format('MMM DD, YYYY')}
								</TableCell>
								<TableCell>
									<div className="flex gap-1">
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openEditDialog(perm)}
										>
											<Pencil className="h-4 w-4" />
										</Button>
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openDeleteDialog(perm)}
										>
											<Trash2 className="h-4 w-4 text-red-500" />
										</Button>
									</div>
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
			) : (
				<div className="flex min-h-[25vh] flex-col items-center justify-center text-gray-300">
					<AlertCircle className="h-16 w-16 mb-2" />
					<p className="text-2xl font-bold">No Permissions</p>
					<p className="text-sm mt-1">
						Create resources, scopes, and policies first, then combine them
						into permissions.
					</p>
				</div>
			)}

			{/* Add/Edit Dialog */}
			<Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
				<DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
					<DialogHeader>
						<DialogTitle>
							{editingPermission ? 'Edit Permission' : 'Add Permission'}
						</DialogTitle>
						<DialogDescription>
							{editingPermission
								? 'Update the permission configuration.'
								: 'Build a permission by combining resource, scopes, and policies.'}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label htmlFor="perm-name">Name</Label>
							<Input
								id="perm-name"
								placeholder="e.g. View Documents"
								value={formName}
								onChange={(e) => setFormName(e.target.value)}
								className="mt-1"
							/>
						</div>
						<div>
							<Label htmlFor="perm-desc">Description</Label>
							<Textarea
								id="perm-desc"
								placeholder="Optional description"
								value={formDescription}
								onChange={(e) => setFormDescription(e.target.value)}
								className="mt-1"
							/>
						</div>

						{/* Resource */}
						<div>
							<Label htmlFor="perm-resource">Resource</Label>
							<Select
								id="perm-resource"
								value={formResourceId}
								onChange={(e) => setFormResourceId(e.target.value)}
								className="mt-1"
								disabled={!!editingPermission}
							>
								<option value="">Select a resource...</option>
								{allResources.map((r) => (
									<option key={r.id} value={r.id}>
										{r.name}
									</option>
								))}
							</Select>
						</div>

						{/* Scopes - multi-select checkboxes */}
						<div>
							<Label>Scopes</Label>
							<div className="mt-1 space-y-2 max-h-32 overflow-y-auto border rounded-md p-2">
								{allScopes.length === 0 ? (
									<p className="text-sm text-gray-400">
										No scopes defined yet.
									</p>
								) : (
									allScopes.map((scope) => (
										<label
											key={scope.id}
											className="flex items-center gap-2 text-sm cursor-pointer"
										>
											<input
												type="checkbox"
												checked={formScopeIds.includes(scope.id)}
												onChange={() => toggleScopeId(scope.id)}
												className="rounded border-gray-300"
											/>
											<span>{scope.name}</span>
											{scope.description && (
												<span className="text-gray-400">
													- {scope.description}
												</span>
											)}
										</label>
									))
								)}
							</div>
						</div>

						{/* Policies - multi-select checkboxes with type preview */}
						<div>
							<Label>Policies</Label>
							<div className="mt-1 space-y-2 max-h-40 overflow-y-auto border rounded-md p-2">
								{allPolicies.length === 0 ? (
									<p className="text-sm text-gray-400">
										No policies defined yet.
									</p>
								) : (
									allPolicies.map((policy) => (
										<label
											key={policy.id}
											className="flex items-center gap-2 text-sm cursor-pointer"
										>
											<input
												type="checkbox"
												checked={formPolicyIds.includes(policy.id)}
												onChange={() => togglePolicyId(policy.id)}
												className="rounded border-gray-300"
											/>
											<span>{policy.name}</span>
											<Badge
												variant={
													policy.type === 'role' ? 'default' : 'secondary'
												}
												className="text-xs"
											>
												{policy.type}
											</Badge>
											<span className="text-gray-400 text-xs">
												{policy.targets
													.map((t) => t.target_value)
													.join(', ')}
											</span>
										</label>
									))
								)}
							</div>
						</div>

						{/* Decision strategy */}
						<div>
							<Label htmlFor="perm-strategy">Decision Strategy</Label>
							<Select
								id="perm-strategy"
								value={formDecisionStrategy}
								onChange={(e) => setFormDecisionStrategy(e.target.value)}
								className="mt-1"
							>
								<option value="affirmative">Any match grants</option>
								<option value="unanimous">All must match</option>
							</Select>
						</div>

						{/* Natural language summary */}
						<div className="rounded-md bg-blue-50 p-3 text-sm text-blue-800">
							<strong>Summary: </strong>
							{buildSummary()}
						</div>
					</div>
					<DialogFooter>
						<Button
							variant="outline"
							onClick={() => setDialogOpen(false)}
							disabled={saving}
						>
							Cancel
						</Button>
						<Button onClick={handleSave} disabled={saving}>
							{saving
								? 'Saving...'
								: editingPermission
									? 'Update'
									: 'Add'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Delete Confirmation Dialog */}
			<Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete Permission</DialogTitle>
						<DialogDescription>
							Are you sure you want to delete{' '}
							<strong>{deletingPermission?.name}</strong>? This action cannot be
							undone.
						</DialogDescription>
					</DialogHeader>
					<DialogFooter>
						<Button
							variant="outline"
							onClick={() => setDeleteDialogOpen(false)}
							disabled={saving}
						>
							Cancel
						</Button>
						<Button
							variant="destructive"
							onClick={handleDelete}
							disabled={saving}
						>
							{saving ? 'Deleting...' : 'Delete'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
