import React from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import { Plus, Pencil, Trash2, AlertCircle } from 'lucide-react';
import { PoliciesQuery } from '../../graphql/queries';
import { AddPolicy, UpdatePolicy, DeletePolicy } from '../../graphql/mutation';
import { getGraphQLErrorMessage } from '../../utils';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Textarea } from '../../components/ui/textarea';
import { Label } from '../../components/ui/label';
import { Badge } from '../../components/ui/badge';
import { Skeleton } from '../../components/ui/skeleton';
import { Select } from '../../components/ui/select';
import { Switch } from '../../components/ui/switch';
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
import type { AuthzPolicy, AuthzPoliciesResponse } from '../../types';

interface PolicyTarget {
	target_type: string;
	target_value: string;
}

export default function Policies() {
	const client = useClient();
	const [policies, setPolicies] = React.useState<AuthzPolicy[]>([]);
	const [loading, setLoading] = React.useState(false);
	const [dialogOpen, setDialogOpen] = React.useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
	const [editingPolicy, setEditingPolicy] = React.useState<AuthzPolicy | null>(
		null,
	);
	const [deletingPolicy, setDeletingPolicy] =
		React.useState<AuthzPolicy | null>(null);
	const [formName, setFormName] = React.useState('');
	const [formDescription, setFormDescription] = React.useState('');
	const [formType, setFormType] = React.useState('role');
	const [formLogic, setFormLogic] = React.useState('positive');
	const [formDecisionStrategy, setFormDecisionStrategy] =
		React.useState('affirmative');
	const [formTargets, setFormTargets] = React.useState<PolicyTarget[]>([
		{ target_type: 'role', target_value: '' },
	]);
	const [saving, setSaving] = React.useState(false);

	const fetchPolicies = async () => {
		setLoading(true);
		const { data, error } = await client
			.query<AuthzPoliciesResponse>(PoliciesQuery, {
				params: { pagination: { limit: 100, page: 1 } },
			})
			.toPromise();
		if (data?._policies) {
			setPolicies(data._policies.policies);
		}
		if (error) {
			toast.error(getGraphQLErrorMessage(error, 'Failed to load policies'));
		}
		setLoading(false);
	};

	React.useEffect(() => {
		fetchPolicies();
	}, []);

	const openAddDialog = () => {
		setEditingPolicy(null);
		setFormName('');
		setFormDescription('');
		setFormType('role');
		setFormLogic('positive');
		setFormDecisionStrategy('affirmative');
		setFormTargets([{ target_type: 'role', target_value: '' }]);
		setDialogOpen(true);
	};

	const openEditDialog = (policy: AuthzPolicy) => {
		setEditingPolicy(policy);
		setFormName(policy.name);
		setFormDescription(policy.description || '');
		setFormType(policy.type);
		setFormLogic(policy.logic);
		setFormDecisionStrategy(policy.decision_strategy);
		setFormTargets(
			policy.targets.map((t) => ({
				target_type: t.target_type,
				target_value: t.target_value,
			})),
		);
		setDialogOpen(true);
	};

	const openDeleteDialog = (policy: AuthzPolicy) => {
		setDeletingPolicy(policy);
		setDeleteDialogOpen(true);
	};

	const handleTypeChange = (newType: string) => {
		setFormType(newType);
		setFormTargets([{ target_type: newType, target_value: '' }]);
	};

	const addTarget = () => {
		setFormTargets([
			...formTargets,
			{ target_type: formType, target_value: '' },
		]);
	};

	const removeTarget = (index: number) => {
		setFormTargets(formTargets.filter((_, i) => i !== index));
	};

	const updateTarget = (index: number, value: string) => {
		const updated = [...formTargets];
		updated[index] = { ...updated[index], target_value: value };
		setFormTargets(updated);
	};

	const handleSave = async () => {
		if (!formName.trim()) {
			toast.error('Name is required');
			return;
		}
		const validTargets = formTargets.filter(
			(t) => t.target_value.trim() !== '',
		);
		if (validTargets.length === 0) {
			toast.error('At least one target is required');
			return;
		}
		setSaving(true);
		if (editingPolicy) {
			const { error } = await client
				.mutation(UpdatePolicy, {
					params: {
						id: editingPolicy.id,
						name: formName,
						description: formDescription || undefined,
						logic: formLogic,
						decision_strategy: formDecisionStrategy,
						targets: validTargets,
					},
				})
				.toPromise();
			if (error) {
				toast.error(getGraphQLErrorMessage(error, 'Failed to update policy'));
			} else {
				toast.success('Policy updated');
				setDialogOpen(false);
				fetchPolicies();
			}
		} else {
			const { error } = await client
				.mutation(AddPolicy, {
					params: {
						name: formName,
						description: formDescription || undefined,
						type: formType,
						logic: formLogic,
						decision_strategy: formDecisionStrategy,
						targets: validTargets,
					},
				})
				.toPromise();
			if (error) {
				toast.error(getGraphQLErrorMessage(error, 'Failed to add policy'));
			} else {
				toast.success('Policy added');
				setDialogOpen(false);
				fetchPolicies();
			}
		}
		setSaving(false);
	};

	const handleDelete = async () => {
		if (!deletingPolicy) return;
		setSaving(true);
		const { error } = await client
			.mutation(DeletePolicy, { id: deletingPolicy.id })
			.toPromise();
		if (error) {
			toast.error(getGraphQLErrorMessage(error, 'Failed to delete policy'));
		} else {
			toast.success('Policy deleted');
			setDeleteDialogOpen(false);
			setDeletingPolicy(null);
			fetchPolicies();
		}
		setSaving(false);
	};

	return (
		<div>
			<div className="flex items-center justify-between mb-4">
				<div>
					<h2 className="text-lg font-semibold text-gray-900">Policies</h2>
					<p className="text-sm text-gray-500">
						Define who gets access based on roles or specific users.
					</p>
				</div>
				<Button onClick={openAddDialog}>
					<Plus className="h-4 w-4 mr-2" />
					Add Policy
				</Button>
			</div>

			{loading ? (
				<div className="space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : policies.length > 0 ? (
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Name</TableHead>
							<TableHead>Type</TableHead>
							<TableHead>Targets</TableHead>
							<TableHead>Logic</TableHead>
							<TableHead>Strategy</TableHead>
							<TableHead>Created</TableHead>
							<TableHead className="w-[100px]">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{policies.map((policy) => (
							<TableRow key={policy.id}>
								<TableCell className="font-medium">{policy.name}</TableCell>
								<TableCell>
									<Badge variant={policy.type === 'role' ? 'default' : 'secondary'}>
										{policy.type}
									</Badge>
								</TableCell>
								<TableCell className="text-sm text-gray-500 max-w-[200px] truncate">
									{policy.targets
										.map((t) => t.target_value)
										.join(', ')}
								</TableCell>
								<TableCell>
									<Badge
										variant={
											policy.logic === 'positive' ? 'success' : 'destructive'
										}
									>
										{policy.logic === 'positive' ? 'Grant' : 'Deny'}
									</Badge>
								</TableCell>
								<TableCell className="text-sm">
									{policy.decision_strategy === 'affirmative'
										? 'Any match'
										: 'All must match'}
								</TableCell>
								<TableCell className="text-sm">
									{dayjs(policy.created_at * 1000).format('MMM DD, YYYY')}
								</TableCell>
								<TableCell>
									<div className="flex gap-1">
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openEditDialog(policy)}
										>
											<Pencil className="h-4 w-4" />
										</Button>
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openDeleteDialog(policy)}
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
					<p className="text-2xl font-bold">No Policies</p>
					<p className="text-sm mt-1">
						Add a policy to define access rules.
					</p>
				</div>
			)}

			{/* Add/Edit Dialog */}
			<Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
				<DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
					<DialogHeader>
						<DialogTitle>
							{editingPolicy ? 'Edit Policy' : 'Add Policy'}
						</DialogTitle>
						<DialogDescription>
							{editingPolicy
								? 'Update the policy configuration.'
								: 'Create a new access policy.'}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label htmlFor="policy-name">Name</Label>
							<Input
								id="policy-name"
								placeholder="e.g. Admin Access, Editor Role"
								value={formName}
								onChange={(e) => setFormName(e.target.value)}
								className="mt-1"
							/>
						</div>
						<div>
							<Label htmlFor="policy-desc">Description</Label>
							<Textarea
								id="policy-desc"
								placeholder="Optional description"
								value={formDescription}
								onChange={(e) => setFormDescription(e.target.value)}
								className="mt-1"
							/>
						</div>

						{/* Type toggle */}
						{!editingPolicy && (
							<div>
								<Label>Type</Label>
								<div className="flex items-center gap-3 mt-1">
									<span
										className={`text-sm ${formType === 'role' ? 'font-semibold text-gray-900' : 'text-gray-500'}`}
									>
										Roles
									</span>
									<Switch
										checked={formType === 'user'}
										onCheckedChange={(checked) =>
											handleTypeChange(checked ? 'user' : 'role')
										}
									/>
									<span
										className={`text-sm ${formType === 'user' ? 'font-semibold text-gray-900' : 'text-gray-500'}`}
									>
										Specific Users
									</span>
								</div>
							</div>
						)}

						{/* Logic */}
						<div>
							<Label htmlFor="policy-logic">Logic</Label>
							<Select
								id="policy-logic"
								value={formLogic}
								onChange={(e) => setFormLogic(e.target.value)}
								className="mt-1"
							>
								<option value="positive">Grant when matched</option>
								<option value="negative">Deny when matched</option>
							</Select>
						</div>

						{/* Decision strategy */}
						<div>
							<Label htmlFor="policy-strategy">Decision Strategy</Label>
							<Select
								id="policy-strategy"
								value={formDecisionStrategy}
								onChange={(e) => setFormDecisionStrategy(e.target.value)}
								className="mt-1"
							>
								<option value="affirmative">Any match grants</option>
								<option value="unanimous">All must match</option>
							</Select>
						</div>

						{/* Targets */}
						<div>
							<Label>
								{formType === 'role' ? 'Role Names' : 'User Identifiers'}
							</Label>
							<div className="space-y-2 mt-1">
								{formTargets.map((target, index) => (
									<div key={index} className="flex gap-2">
										<Input
											placeholder={
												formType === 'role'
													? 'e.g. admin, editor'
													: 'e.g. user@example.com'
											}
											value={target.target_value}
											onChange={(e) => updateTarget(index, e.target.value)}
											className="flex-1"
										/>
										{formTargets.length > 1 && (
											<Button
												variant="ghost"
												size="icon"
												onClick={() => removeTarget(index)}
											>
												<Trash2 className="h-4 w-4 text-red-500" />
											</Button>
										)}
									</div>
								))}
								<Button
									variant="outline"
									size="sm"
									onClick={addTarget}
									className="mt-1"
								>
									<Plus className="h-3 w-3 mr-1" />
									Add {formType === 'role' ? 'Role' : 'User'}
								</Button>
							</div>
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
							{saving ? 'Saving...' : editingPolicy ? 'Update' : 'Add'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Delete Confirmation Dialog */}
			<Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete Policy</DialogTitle>
						<DialogDescription>
							Are you sure you want to delete{' '}
							<strong>{deletingPolicy?.name}</strong>? This action cannot be
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
