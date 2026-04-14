import React from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import { Plus, Pencil, Trash2, AlertCircle } from 'lucide-react';
import { ResourcesQuery } from '../../graphql/queries';
import {
	AddResource,
	UpdateResource,
	DeleteResource,
} from '../../graphql/mutation';
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
import type { AuthzResource, AuthzResourcesResponse } from '../../types';

export default function Resources() {
	const client = useClient();
	const [resources, setResources] = React.useState<AuthzResource[]>([]);
	const [loading, setLoading] = React.useState(false);
	const [dialogOpen, setDialogOpen] = React.useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
	const [editingResource, setEditingResource] =
		React.useState<AuthzResource | null>(null);
	const [deletingResource, setDeletingResource] =
		React.useState<AuthzResource | null>(null);
	const [formName, setFormName] = React.useState('');
	const [formDescription, setFormDescription] = React.useState('');
	const [saving, setSaving] = React.useState(false);

	const fetchResources = async () => {
		setLoading(true);
		const { data, error } = await client
			.query<AuthzResourcesResponse>(ResourcesQuery, {
				params: { pagination: { limit: 100, page: 1 } },
			})
			.toPromise();
		if (data?._resources) {
			setResources(data._resources.resources);
		}
		if (error) {
			toast.error(getGraphQLErrorMessage(error, 'Failed to load resources'));
		}
		setLoading(false);
	};

	React.useEffect(() => {
		fetchResources();
	}, []);

	const openAddDialog = () => {
		setEditingResource(null);
		setFormName('');
		setFormDescription('');
		setDialogOpen(true);
	};

	const openEditDialog = (resource: AuthzResource) => {
		setEditingResource(resource);
		setFormName(resource.name);
		setFormDescription(resource.description || '');
		setDialogOpen(true);
	};

	const openDeleteDialog = (resource: AuthzResource) => {
		setDeletingResource(resource);
		setDeleteDialogOpen(true);
	};

	const handleSave = async () => {
		if (!formName.trim()) {
			toast.error('Name is required');
			return;
		}
		setSaving(true);
		if (editingResource) {
			const { error } = await client
				.mutation(UpdateResource, {
					params: {
						id: editingResource.id,
						name: formName,
						description: formDescription || undefined,
					},
				})
				.toPromise();
			if (error) {
				toast.error(
					getGraphQLErrorMessage(error, 'Failed to update resource'),
				);
			} else {
				toast.success('Resource updated');
				setDialogOpen(false);
				fetchResources();
			}
		} else {
			const { error } = await client
				.mutation(AddResource, {
					params: {
						name: formName,
						description: formDescription || undefined,
					},
				})
				.toPromise();
			if (error) {
				toast.error(getGraphQLErrorMessage(error, 'Failed to add resource'));
			} else {
				toast.success('Resource added');
				setDialogOpen(false);
				fetchResources();
			}
		}
		setSaving(false);
	};

	const handleDelete = async () => {
		if (!deletingResource) return;
		setSaving(true);
		const { error } = await client
			.mutation(DeleteResource, { id: deletingResource.id })
			.toPromise();
		if (error) {
			toast.error(getGraphQLErrorMessage(error, 'Failed to delete resource'));
		} else {
			toast.success('Resource deleted');
			setDeleteDialogOpen(false);
			setDeletingResource(null);
			fetchResources();
		}
		setSaving(false);
	};

	return (
		<div>
			<div className="flex items-center justify-between mb-4">
				<div>
					<h2 className="text-lg font-semibold text-gray-900">Resources</h2>
					<p className="text-sm text-gray-500">
						Define the resources to protect with fine-grained authorization.
					</p>
				</div>
				<Button onClick={openAddDialog}>
					<Plus className="h-4 w-4 mr-2" />
					Add Resource
				</Button>
			</div>

			{loading ? (
				<div className="space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : resources.length > 0 ? (
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
						{resources.map((resource) => (
							<TableRow key={resource.id}>
								<TableCell className="font-medium">
									{resource.name}
								</TableCell>
								<TableCell className="text-sm text-gray-500">
									{resource.description || '-'}
								</TableCell>
								<TableCell className="text-sm">
									{dayjs(resource.created_at * 1000).format('MMM DD, YYYY')}
								</TableCell>
								<TableCell>
									<div className="flex gap-1">
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openEditDialog(resource)}
										>
											<Pencil className="h-4 w-4" />
										</Button>
										<Button
											variant="ghost"
											size="icon"
											onClick={() => openDeleteDialog(resource)}
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
					<p className="text-2xl font-bold">No Resources</p>
					<p className="text-sm mt-1">
						Add a resource to get started with authorization.
					</p>
				</div>
			)}

			{/* Add/Edit Dialog */}
			<Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>
							{editingResource ? 'Edit Resource' : 'Add Resource'}
						</DialogTitle>
						<DialogDescription>
							{editingResource
								? 'Update the resource details.'
								: 'Define a new resource to protect.'}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label htmlFor="resource-name">Name</Label>
							<Input
								id="resource-name"
								placeholder="e.g. documents, projects"
								value={formName}
								onChange={(e) => setFormName(e.target.value)}
								className="mt-1"
							/>
						</div>
						<div>
							<Label htmlFor="resource-desc">Description</Label>
							<Textarea
								id="resource-desc"
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
							{saving ? 'Saving...' : editingResource ? 'Update' : 'Add'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Delete Confirmation Dialog */}
			<Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete Resource</DialogTitle>
						<DialogDescription>
							Are you sure you want to delete{' '}
							<strong>{deletingResource?.name}</strong>? This action cannot be
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
