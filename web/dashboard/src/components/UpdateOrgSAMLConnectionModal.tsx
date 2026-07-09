import React, { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { UpdateModalViews } from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import {
	CreateOrgSAMLConnection,
	UpdateOrgSAMLConnection,
} from '../graphql/mutation';
import type { OrgSAMLConnection } from '../types';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Switch } from './ui/switch';
import { Textarea } from './ui/textarea';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';

interface UpdateOrgSAMLConnectionModalProps {
	view: UpdateModalViews;
	orgId: string;
	selectedConnection?: OrgSAMLConnection;
	fetchConnection: () => void;
}

interface SAMLFormData {
	name: string;
	idpEntityId: string;
	idpSsoUrl: string;
	// Required at creation; on edit, filling it replaces the stored certificate.
	idpCertificate: string;
	spEntityId: string;
	acsUrl: string;
	attributeMapping: string;
	allowIdpInitiated: boolean;
	isActive: boolean;
}

const initFormData: SAMLFormData = {
	name: '',
	idpEntityId: '',
	idpSsoUrl: '',
	idpCertificate: '',
	spEntityId: '',
	acsUrl: '',
	attributeMapping: '',
	allowIdpInitiated: false,
	isActive: true,
};

const UpdateOrgSAMLConnectionModal = ({
	view,
	orgId,
	selectedConnection,
	fetchConnection,
}: UpdateOrgSAMLConnectionModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [formData, setFormData] = useState<SAMLFormData>({ ...initFormData });

	const isEdit = view === UpdateModalViews.Edit;

	useEffect(() => {
		if (open && isEdit && selectedConnection) {
			setFormData({
				name: selectedConnection.name,
				idpEntityId: selectedConnection.idp_entity_id,
				idpSsoUrl: selectedConnection.idp_sso_url || '',
				idpCertificate: '',
				spEntityId: selectedConnection.sp_entity_id || '',
				acsUrl: selectedConnection.acs_url || '',
				attributeMapping: selectedConnection.attribute_mapping || '',
				allowIdpInitiated: selectedConnection.allow_idp_initiated,
				isActive: selectedConnection.is_active,
			});
		}
	}, [open]);

	const setField = (field: keyof SAMLFormData, value: string | boolean) =>
		setFormData({ ...formData, [field]: value });

	const validateData = () => {
		if (loading) return false;
		if (isEdit) {
			return (
				formData.name.trim().length > 0 &&
				formData.idpEntityId.trim().length > 0
			);
		}
		return (
			formData.name.trim().length > 0 &&
			formData.idpEntityId.trim().length > 0 &&
			formData.idpSsoUrl.trim().length > 0 &&
			formData.idpCertificate.trim().length > 0
		);
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let res: { error?: unknown };
		if (isEdit && selectedConnection?.id) {
			res = await client
				.mutation(UpdateOrgSAMLConnection, {
					params: {
						id: selectedConnection.id,
						name: formData.name.trim(),
						idp_entity_id: formData.idpEntityId.trim(),
						idp_sso_url: formData.idpSsoUrl.trim() || undefined,
						// Omitting idp_certificate leaves the stored cert intact.
						idp_certificate: formData.idpCertificate.trim() || undefined,
						sp_entity_id: formData.spEntityId.trim() || undefined,
						acs_url: formData.acsUrl.trim() || undefined,
						attribute_mapping: formData.attributeMapping.trim() || undefined,
						allow_idp_initiated: formData.allowIdpInitiated,
						is_active: formData.isActive,
					},
				})
				.toPromise();
		} else {
			res = await client
				.mutation(CreateOrgSAMLConnection, {
					params: {
						org_id: orgId,
						name: formData.name.trim(),
						idp_entity_id: formData.idpEntityId.trim(),
						idp_sso_url: formData.idpSsoUrl.trim(),
						idp_certificate: formData.idpCertificate.trim(),
						sp_entity_id: formData.spEntityId.trim() || undefined,
						acs_url: formData.acsUrl.trim() || undefined,
						attribute_mapping: formData.attributeMapping.trim() || undefined,
						allow_idp_initiated: formData.allowIdpInitiated,
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
							? 'Failed to update SAML connection'
							: 'Failed to create SAML connection',
					),
				),
			);
			return;
		}
		toast.success(
			isEdit ? 'SAML connection updated' : 'SAML connection created',
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
					Configure SAML
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
								? 'Configure SAML Connection'
								: 'Edit SAML Connection'}
						</SheetTitle>
						<SheetDescription>
							Upstream SAML 2.0 identity provider for this organization.
							Authorizer acts as the Service Provider. The IdP certificate is
							never shown again after saving.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">Name</label>
							<Input
								placeholder="okta-saml"
								value={formData.name}
								isInvalid={formData.name.trim().length === 0}
								onChange={(e) => setField('name', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								IdP Entity ID
							</label>
							<Input
								placeholder="https://idp.example.com/saml/metadata"
								value={formData.idpEntityId}
								isInvalid={formData.idpEntityId.trim().length === 0}
								onChange={(e) => setField('idpEntityId', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								IdP SSO URL
							</label>
							<Input
								placeholder="https://idp.example.com/saml/sso"
								value={formData.idpSsoUrl}
								isInvalid={!isEdit && formData.idpSsoUrl.trim().length === 0}
								onChange={(e) => setField('idpSsoUrl', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-start gap-4">
							<label className="w-32 text-sm font-medium shrink-0 pt-2">
								IdP Certificate
							</label>
							<Textarea
								rows={5}
								className="font-mono text-xs"
								placeholder={
									isEdit
										? 'Leave empty to keep the current certificate'
										: '-----BEGIN CERTIFICATE----- (PEM)'
								}
								value={formData.idpCertificate}
								onChange={(e) =>
									setField('idpCertificate', e.currentTarget.value)
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								SP Entity ID
							</label>
							<Input
								placeholder="Derived from request host when empty"
								value={formData.spEntityId}
								onChange={(e) => setField('spEntityId', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								ACS URL
							</label>
							<Input
								placeholder="Derived from request host when empty"
								value={formData.acsUrl}
								onChange={(e) => setField('acsUrl', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-start gap-4">
							<label className="w-32 text-sm font-medium shrink-0 pt-2">
								Attribute Mapping
							</label>
							<Textarea
								rows={3}
								className="font-mono text-xs"
								placeholder='{"email":"email","given_name":"firstName"}'
								value={formData.attributeMapping}
								onChange={(e) =>
									setField('attributeMapping', e.currentTarget.value)
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-32 text-sm font-medium shrink-0">
								Allow IdP Initiated
							</label>
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium">Off</span>
								<Switch
									checked={formData.allowIdpInitiated}
									onCheckedChange={(checked: boolean) =>
										setField('allowIdpInitiated', checked)
									}
								/>
								<span className="text-sm font-medium">On</span>
							</div>
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

export default UpdateOrgSAMLConnectionModal;
