import React, { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { UpdateModalViews } from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import { CreateOrganization, UpdateOrganization } from '../graphql/mutation';
import type { Organization } from '../types';
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

interface UpdateOrganizationModalProps {
	view: UpdateModalViews;
	selectedOrganization?: Organization;
	fetchOrganizations: () => void;
}

interface OrganizationFormData {
	name: string;
	displayName: string;
	enabled: boolean;
}

const initFormData: OrganizationFormData = {
	name: '',
	displayName: '',
	enabled: true,
};

const UpdateOrganizationModal = ({
	view,
	selectedOrganization,
	fetchOrganizations,
}: UpdateOrganizationModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [formData, setFormData] = useState<OrganizationFormData>({
		...initFormData,
	});

	const isEdit = view === UpdateModalViews.Edit;

	useEffect(() => {
		if (open && isEdit && selectedOrganization) {
			setFormData({
				name: selectedOrganization.name,
				displayName: selectedOrganization.display_name || '',
				enabled: selectedOrganization.enabled,
			});
		}
	}, [open]);

	const validateData = () => !loading && formData.name.trim().length > 0;

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let res: { error?: unknown };
		if (isEdit && selectedOrganization?.id) {
			res = await client
				.mutation(UpdateOrganization, {
					params: {
						id: selectedOrganization.id,
						name: formData.name.trim(),
						display_name: formData.displayName,
						enabled: formData.enabled,
					},
				})
				.toPromise();
		} else {
			res = await client
				.mutation(CreateOrganization, {
					params: {
						name: formData.name.trim(),
						display_name: formData.displayName,
					},
				})
				.toPromise();
		}
		setLoading(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(
						res.error,
						isEdit
							? 'Failed to update organization'
							: 'Failed to create organization',
					),
				),
			);
			return;
		}
		toast.success(isEdit ? 'Organization updated' : 'Organization created');
		setFormData({ ...initFormData });
		setOpen(false);
		fetchOrganizations();
	};

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button size="sm" onClick={() => setOpen(true)}>
					<Plus className="mr-2 h-4 w-4" />
					Add Organization
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
							{view === UpdateModalViews.ADD
								? 'Add New Organization'
								: 'Edit Organization'}
						</SheetTitle>
						<SheetDescription>
							Organizations group users for per-org SSO connections and SCIM
							provisioning.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">Name</label>
							<Input
								placeholder="acme-corp (unique, URL-safe slug)"
								value={formData.name}
								isInvalid={formData.name.trim().length === 0}
								onChange={(e) =>
									setFormData({ ...formData, name: e.currentTarget.value })
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Display Name
							</label>
							<Input
								placeholder="Acme Corp"
								value={formData.displayName}
								onChange={(e) =>
									setFormData({
										...formData,
										displayName: e.currentTarget.value,
									})
								}
							/>
						</div>

						{isEdit && (
							<div className="flex items-center gap-4">
								<label className="w-32 text-sm font-medium shrink-0">
									Enabled
								</label>
								<div className="flex items-center gap-2">
									<span className="text-sm font-medium">Off</span>
									<Switch
										checked={formData.enabled}
										onCheckedChange={(checked: boolean) =>
											setFormData({ ...formData, enabled: checked })
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
		</>
	);
};

export default UpdateOrganizationModal;
