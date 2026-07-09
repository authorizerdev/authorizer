import React, { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { UpdateModalViews } from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import { CreateClient, UpdateClient } from '../graphql/mutation';
import type { Client, CreateClientResponse } from '../types';
import ClientSecretDialog from './ClientSecretDialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Switch } from './ui/switch';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';

interface UpdateClientModalProps {
	view: UpdateModalViews;
	selectedClient?: Client;
	fetchClients: () => void;
}

interface ClientFormData {
	name: string;
	description: string;
	// Raw comma/space separated scopes input; parsed on save.
	allowedScopes: string;
	isActive: boolean;
}

const initFormData: ClientFormData = {
	name: '',
	description: '',
	allowedScopes: '',
	isActive: true,
};

const parseScopes = (value: string): string[] =>
	value
		.split(/[\s,]+/)
		.map((scope) => scope.trim())
		.filter(Boolean);

const UpdateClientModal = ({
	view,
	selectedClient,
	fetchClients,
}: UpdateClientModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [formData, setFormData] = useState<ClientFormData>({ ...initFormData });
	// Plaintext secret returned once by _create_client; shown in one-time dialog.
	const [createdSecret, setCreatedSecret] = useState<string | null>(null);

	useEffect(() => {
		if (open && view === UpdateModalViews.Edit && selectedClient) {
			setFormData({
				name: selectedClient.name,
				description: selectedClient.description || '',
				allowedScopes: selectedClient.allowed_scopes.join(' '),
				isActive: selectedClient.is_active,
			});
		}
	}, [open]);

	const validateData = () =>
		!loading &&
		formData.name.trim().length > 0 &&
		parseScopes(formData.allowedScopes).length > 0;

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		if (view === UpdateModalViews.Edit && selectedClient?.id) {
			const res = await client
				.mutation(UpdateClient, {
					params: {
						id: selectedClient.id,
						name: formData.name.trim(),
						description: formData.description,
						allowed_scopes: parseScopes(formData.allowedScopes),
						is_active: formData.isActive,
					},
				})
				.toPromise();
			setLoading(false);
			if (res.error) {
				toast.error(
					capitalizeFirstLetter(
						getGraphQLErrorMessage(res.error, 'Failed to update client'),
					),
				);
				return;
			}
			toast.success('Client updated');
		} else {
			const res = await client
				.mutation<{ _create_client: CreateClientResponse }>(CreateClient, {
					params: {
						name: formData.name.trim(),
						description: formData.description,
						allowed_scopes: parseScopes(formData.allowedScopes),
					},
				})
				.toPromise();
			setLoading(false);
			if (res.error) {
				toast.error(
					capitalizeFirstLetter(
						getGraphQLErrorMessage(res.error, 'Failed to create client'),
					),
				);
				return;
			}
			toast.success('Client created');
			setCreatedSecret(res.data?._create_client?.client_secret || null);
		}
		setFormData({ ...initFormData });
		setOpen(false);
		fetchClients();
	};

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button size="sm" onClick={() => setOpen(true)}>
					<Plus className="mr-2 h-4 w-4" />
					Add Client
				</Button>
			) : (
				<button
					className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm"
					onClick={() => setOpen(true)}
				>
					Edit
				</button>
			)}
			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>
							{view === UpdateModalViews.ADD ? 'Add New Client' : 'Edit Client'}
						</SheetTitle>
						<SheetDescription>
							OAuth clients are machine/workload identities with their own
							credentials and allowed scopes.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">Name</label>
							<Input
								placeholder="my-service"
								value={formData.name}
								isInvalid={formData.name.trim().length === 0}
								onChange={(e) =>
									setFormData({ ...formData, name: e.currentTarget.value })
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Description
							</label>
							<Input
								placeholder="What this client is used for"
								value={formData.description}
								onChange={(e) =>
									setFormData({
										...formData,
										description: e.currentTarget.value,
									})
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Allowed Scopes
							</label>
							<Input
								placeholder="read:users write:users (space or comma separated)"
								value={formData.allowedScopes}
								isInvalid={parseScopes(formData.allowedScopes).length === 0}
								onChange={(e) =>
									setFormData({
										...formData,
										allowedScopes: e.currentTarget.value,
									})
								}
							/>
						</div>

						{view === UpdateModalViews.Edit && (
							<div className="flex items-center gap-4">
								<label className="w-32 text-sm font-medium shrink-0">
									Active
								</label>
								<div className="flex items-center gap-2">
									<span className="text-sm font-medium">Off</span>
									<Switch
										checked={formData.isActive}
										onCheckedChange={(checked: boolean) =>
											setFormData({ ...formData, isActive: checked })
										}
									/>
									<span className="text-sm font-medium">On</span>
								</div>
							</div>
						)}
					</div>

					<SheetFooter className="mt-6">
						<Button
							onClick={saveData}
							isLoading={loading}
							disabled={!validateData()}
						>
							Save
						</Button>
					</SheetFooter>
				</SheetContent>
			</Sheet>
			<ClientSecretDialog
				secret={createdSecret}
				onClose={() => setCreatedSecret(null)}
			/>
		</>
	);
};

export default UpdateClientModal;
