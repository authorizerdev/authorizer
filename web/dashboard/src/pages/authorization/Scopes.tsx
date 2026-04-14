import React from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import { Plus, Pencil, Trash2, AlertCircle } from 'lucide-react';
import { ScopesQuery } from '../../graphql/queries';
import { AddScope, UpdateScope, DeleteScope } from '../../graphql/mutation';
import { getGraphQLErrorMessage } from '../../utils';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Textarea } from '../../components/ui/textarea';
import { Label } from '../../components/ui/label';
import { Skeleton } from '../../components/ui/skeleton';
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
import type { AuthzScope, AuthzScopesResponse } from '../../types';

export default function Scopes() {
	const client = useClient();
	const [scopes, setScopes] = React.useState<AuthzScope[]>([]);
	const [loading, setLoading] = React.useState(false);
	const [dialogOpen, setDialogOpen] = React.useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
	const [editingScope, setEditingScope] = React.useState<AuthzScope | null>(
		null,
	);
	const [deletingScope, setDeletingScope] = React.useState<AuthzScope | null>(
		null,
	);
	const [formName, setFormName] = React.useState('');
	const [formDescription, setFormDescription] = React.useState('');
	const [saving, setSaving] = React.useState(false);

	const fetchScopes = async () => {
		setLoading(true);
		const { data, error } = await client
			.query<AuthzScopesResponse>(ScopesQuery, {
				params: { pagination: { limit: 100, page: 1 } },
			})
			.toPromise();
		if (data?._scopes) {
			setScopes(data._scopes.scopes);
		}
		if (error) {
			toast.error(getGraphQLErrorMessage(error, 'Failed to load scopes'));
		}
		setLoading(false);
	};

	React.useEffect(() => {
		fetchScopes();
	}, []);

	const openAddDialog = () => {
		setEditingScope(null);
		setFormName('');
		setFormDescription('');
		setDialogOpen(true);
	};

	const openEditDialog = (scope: AuthzScope) => {
		setEditingScope(scope);
		setFormName(scope.name);
		setFormDescription(scope.description || '');
		setDialogOpen(true);
	};

	const openDeleteDialog = (scope: AuthzScope) => {
		setDeletingScope(scope);
		setDeleteDialogOpen(true);
	};

	const handleSave = async () => {
		if (!formName.trim()) {
			toast.error('Name is required');
			return;
		}
		setSaving(true);
		if (editingScope) {
			const { error } = await client
				.mutation(UpdateScope, {
					params: {
						id: editingScope.id,
						name: formName,
						description: formDescription || undefined,
					},
				})
				.toPromise();
			if (error) {
				toast.error(getGraphQLErrorMessage(error, 'Failed to update scope'));
			} else {
				toast.success('Scope updated');
				setDialogOpen(false);
				fetchScopes();
			}
		} else {
			const { error } = await client
				.mutation(AddScope, {
					params: {
						name: formName,
						description: formDescription || undefined,
					},
				})
				.toPromise();
			if (error) {
				toast.error(getGraphQLErrorMessage(error, 'Failed to add scope'));
			} else {
				toast.success('Scope added');
				setDialogOpen(false);
				fetchScopes();
			}
		}
		setSaving(false);
	};

	const handleDelete = async () => {
		if (!deletingScope) return;
		setSaving(true);
		const { error } = await client
			.mutation(DeleteScope, { id: deletingScope.id })
			.toPromise();
		if (error) {
			toast.error(getGraphQLErrorMessage(error, 'Failed to delete scope'));
		} else {
			toast.success('Scope deleted');
			setDeleteDialogOpen(false);
			setDeletingScope(null);
			fetchScopes();
		}
		setSaving(false);
	};

	return (
		<div>
			<div className="flex items-center justify-between mb-4">
				<div>
					<h2 className="text-lg font-semibold text-gray-900">Scopes</h2>
					<p className="text-sm text-gray-500">
						Define the actions or operations that can be performed on resources.
					</p>
				</div>
				<Button onClick={openAddDialog}>
					<Plus className="h-4 w-4 mr-2" />
					Add Scope
				</Button>
			</div>

			{loading ? (
				<div className="space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : scopes.length > 0 ? (
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Name</TableHead>
							<TableHead>Description</TableHead>
							<TableHead>Created</TableHead>
							<TableHead className="w-[100px]">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{scopes.map((scope) => (
							<TableRow key={scope.id}>
								<TableCell className="font-medium">{scope.name}</TableCell>
								<TableCell className="text-sm text-gray-500">
									{scope.description || '-'}
								</TableCell>
								<TableCell className="text-sm">
									{dayjs(scope.created_at * 1000).format('MMM DD, YYYY')}
								</TableCell>
								<TableCell>
									<div className="flex gap-1">
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openEditDialog(scope)}
										>
											<Pencil className="h-4 w-4" />
										</Button>
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openDeleteDialog(scope)}
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
					<p className="text-2xl font-bold">No Scopes</p>
					<p className="text-sm mt-1">
						Add a scope to define actions on your resources.
					</p>
				</div>
			)}

			{/* Add/Edit Dialog */}
			<Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>
							{editingScope ? 'Edit Scope' : 'Add Scope'}
						</DialogTitle>
						<DialogDescription>
							{editingScope
								? 'Update the scope details.'
								: 'Define a new scope (action) for authorization.'}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label htmlFor="scope-name">Name</Label>
							<Input
								id="scope-name"
								placeholder="e.g. read, write, delete"
								value={formName}
								onChange={(e) => setFormName(e.target.value)}
								className="mt-1"
							/>
						</div>
						<div>
							<Label htmlFor="scope-desc">Description</Label>
							<Textarea
								id="scope-desc"
								placeholder="Optional description"
								value={formDescription}
								onChange={(e) => setFormDescription(e.target.value)}
								className="mt-1"
							/>
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
							{saving ? 'Saving...' : editingScope ? 'Update' : 'Add'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Delete Confirmation Dialog */}
			<Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete Scope</DialogTitle>
						<DialogDescription>
							Are you sure you want to delete{' '}
							<strong>{deletingScope?.name}</strong>? This action cannot be
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
