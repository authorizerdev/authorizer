import React, { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { UpdateModalViews } from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import {
	CreateOrgOIDCConnection,
	UpdateOrgOIDCConnection,
} from '../graphql/mutation';
import type { OrgOIDCConnection } from '../types';
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

interface UpdateOrgOIDCConnectionModalProps {
	view: UpdateModalViews;
	orgId: string;
	selectedConnection?: OrgOIDCConnection;
	fetchConnection: () => void;
}

interface OIDCFormData {
	name: string;
	issuerUrl: string;
	clientId: string;
	// Required at creation; on edit, filling it rotates the stored secret.
	clientSecret: string;
	scopes: string;
	redirectUri: string;
	isActive: boolean;
}

const initFormData: OIDCFormData = {
	name: '',
	issuerUrl: '',
	clientId: '',
	clientSecret: '',
	scopes: '',
	redirectUri: '',
	isActive: true,
};

const UpdateOrgOIDCConnectionModal = ({
	view,
	orgId,
	selectedConnection,
	fetchConnection,
}: UpdateOrgOIDCConnectionModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [formData, setFormData] = useState<OIDCFormData>({ ...initFormData });

	const isEdit = view === UpdateModalViews.Edit;

	useEffect(() => {
		if (open && isEdit && selectedConnection) {
			setFormData({
				name: selectedConnection.name,
				issuerUrl: selectedConnection.issuer_url,
				clientId: selectedConnection.sso_client_id,
				clientSecret: '',
				scopes: selectedConnection.scopes || '',
				redirectUri: selectedConnection.redirect_uri || '',
				isActive: selectedConnection.is_active,
			});
		}
	}, [open]);

	const setField = (field: keyof OIDCFormData, value: string | boolean) =>
		setFormData({ ...formData, [field]: value });

	const validateData = () => {
		if (loading) return false;
		if (isEdit) {
			return (
				formData.name.trim().length > 0 &&
				formData.issuerUrl.trim().length > 0 &&
				formData.clientId.trim().length > 0
			);
		}
		return (
			formData.name.trim().length > 0 &&
			formData.issuerUrl.trim().length > 0 &&
			formData.clientId.trim().length > 0 &&
			formData.clientSecret.trim().length > 0
		);
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let res: { error?: unknown };
		if (isEdit && selectedConnection?.id) {
			res = await client
				.mutation(UpdateOrgOIDCConnection, {
					params: {
						id: selectedConnection.id,
						name: formData.name.trim(),
						issuer_url: formData.issuerUrl.trim(),
						client_id: formData.clientId.trim(),
						// Omitting client_secret leaves the stored secret intact.
						client_secret: formData.clientSecret.trim() || undefined,
						scopes: formData.scopes.trim() || undefined,
						redirect_uri: formData.redirectUri.trim() || undefined,
						is_active: formData.isActive,
					},
				})
				.toPromise();
		} else {
			res = await client
				.mutation(CreateOrgOIDCConnection, {
					params: {
						org_id: orgId,
						name: formData.name.trim(),
						issuer_url: formData.issuerUrl.trim(),
						client_id: formData.clientId.trim(),
						client_secret: formData.clientSecret.trim(),
						scopes: formData.scopes.trim() || undefined,
						redirect_uri: formData.redirectUri.trim() || undefined,
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
							? 'Failed to update OIDC connection'
							: 'Failed to create OIDC connection',
					),
				),
			);
			return;
		}
		toast.success(
			isEdit ? 'OIDC connection updated' : 'OIDC connection created',
		);
		setFormData({ ...initFormData });
		setOpen(false);
		fetchConnection();
	};

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button size="sm" onClick={() => setOpen(true)}>
					<Plus className="mr-2 h-4 w-4" />
					Configure OIDC
				</Button>
			) : (
				<Button variant="outline" size="sm" onClick={() => setOpen(true)}>
					Edit
				</Button>
			)}
			<Sheet open={open} onOpenChange={setOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>
							{view === UpdateModalViews.ADD
								? 'Configure OIDC Connection'
								: 'Edit OIDC Connection'}
						</SheetTitle>
						<SheetDescription>
							Upstream OIDC identity provider brokered for this organization.
							The client secret is stored encrypted and never shown again.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">Name</label>
							<Input
								placeholder="okta-prod"
								value={formData.name}
								isInvalid={formData.name.trim().length === 0}
								onChange={(e) => setField('name', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Issuer URL
							</label>
							<Input
								placeholder="https://idp.example.com"
								value={formData.issuerUrl}
								isInvalid={formData.issuerUrl.trim().length === 0}
								onChange={(e) => setField('issuerUrl', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Client ID
							</label>
							<Input
								placeholder="Client ID at the upstream IdP"
								value={formData.clientId}
								isInvalid={formData.clientId.trim().length === 0}
								onChange={(e) => setField('clientId', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Client Secret
							</label>
							<Input
								type="password"
								placeholder={
									isEdit
										? 'Leave empty to keep the current secret'
										: 'Client secret at the upstream IdP'
								}
								value={formData.clientSecret}
								isInvalid={!isEdit && formData.clientSecret.trim().length === 0}
								onChange={(e) =>
									setField('clientSecret', e.currentTarget.value)
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Scopes
							</label>
							<Input
								placeholder="openid profile email (space separated)"
								value={formData.scopes}
								onChange={(e) => setField('scopes', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Redirect URI
							</label>
							<Input
								placeholder="Derived from request host when empty"
								value={formData.redirectUri}
								onChange={(e) => setField('redirectUri', e.currentTarget.value)}
							/>
						</div>

						{isEdit && (
							<div className="flex items-center gap-4">
								<label className="w-32 text-sm font-medium shrink-0">
									Active
								</label>
								<div className="flex items-center gap-2">
									<span className="text-sm font-medium">Off</span>
									<Switch
										checked={formData.isActive}
										onCheckedChange={(checked: boolean) =>
											setField('isActive', checked)
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

export default UpdateOrgOIDCConnectionModal;
