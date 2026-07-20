import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import {
	CreateSAMLServiceProvider,
	UpdateSAMLServiceProvider,
} from '../graphql/mutation';
import type { SAMLServiceProvider, SAMLSPMetadataParseResult } from '../types';
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

interface UpdateSAMLServiceProviderModalProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	orgId: string;
	// selectedProvider present => edit; absent => create.
	selectedProvider?: SAMLServiceProvider | null;
	// prefill seeds the create form (e.g. from imported SP metadata).
	prefill?: SAMLSPMetadataParseResult | null;
	onSaved: () => void;
}

interface SPFormData {
	name: string;
	entityId: string;
	acsUrl: string;
	spCertPem: string;
	nameIdFormat: string;
	mappedAttributes: string;
	allowIdpInitiated: boolean;
	isActive: boolean;
}

const initFormData: SPFormData = {
	name: '',
	entityId: '',
	acsUrl: '',
	spCertPem: '',
	nameIdFormat: '',
	mappedAttributes: '',
	allowIdpInitiated: false,
	isActive: true,
};

const UpdateSAMLServiceProviderModal = ({
	open,
	onOpenChange,
	orgId,
	selectedProvider,
	prefill,
	onSaved,
}: UpdateSAMLServiceProviderModalProps) => {
	const client = useClient();
	const [loading, setLoading] = useState(false);
	const [formData, setFormData] = useState<SPFormData>({ ...initFormData });

	const isEdit = !!selectedProvider;

	useEffect(() => {
		if (!open) return;
		if (selectedProvider) {
			setFormData({
				name: selectedProvider.name,
				entityId: selectedProvider.entity_id,
				acsUrl: selectedProvider.acs_url,
				spCertPem: selectedProvider.sp_cert_pem || '',
				nameIdFormat: selectedProvider.name_id_format || '',
				mappedAttributes: selectedProvider.mapped_attributes || '',
				allowIdpInitiated: selectedProvider.allow_idp_initiated,
				isActive: selectedProvider.is_active,
			});
		} else {
			setFormData({
				...initFormData,
				entityId: prefill?.entity_id || '',
				acsUrl: prefill?.acs_url || '',
				spCertPem: prefill?.certificate || '',
			});
		}
	}, [open]);

	const setField = (field: keyof SPFormData, value: string | boolean) =>
		setFormData({ ...formData, [field]: value });

	const validateData = () => {
		if (loading) return false;
		return (
			formData.name.trim().length > 0 &&
			formData.entityId.trim().length > 0 &&
			formData.acsUrl.trim().length > 0
		);
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let res: { error?: unknown };
		if (isEdit && selectedProvider?.id) {
			res = await client
				.mutation(UpdateSAMLServiceProvider, {
					params: {
						id: selectedProvider.id,
						name: formData.name.trim(),
						entity_id: formData.entityId.trim(),
						acs_url: formData.acsUrl.trim(),
						sp_cert_pem: formData.spCertPem.trim() || undefined,
						name_id_format: formData.nameIdFormat.trim() || undefined,
						mapped_attributes: formData.mappedAttributes.trim() || undefined,
						allow_idp_initiated: formData.allowIdpInitiated,
						is_active: formData.isActive,
					},
				})
				.toPromise();
		} else {
			res = await client
				.mutation(CreateSAMLServiceProvider, {
					params: {
						org_id: orgId,
						name: formData.name.trim(),
						entity_id: formData.entityId.trim(),
						acs_url: formData.acsUrl.trim(),
						sp_cert_pem: formData.spCertPem.trim() || undefined,
						name_id_format: formData.nameIdFormat.trim() || undefined,
						mapped_attributes: formData.mappedAttributes.trim() || undefined,
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
							? 'Failed to update SAML service provider'
							: 'Failed to create SAML service provider',
					),
				),
			);
			return;
		}
		toast.success(
			isEdit
				? 'SAML service provider updated'
				: 'SAML service provider created',
		);
		onOpenChange(false);
		onSaved();
	};

	return (
		<Sheet open={open} onOpenChange={onOpenChange}>
			<SheetContent className="overflow-y-auto sm:max-w-2xl">
				<SheetHeader>
					<SheetTitle>
						{isEdit
							? 'Edit SAML Service Provider'
							: 'Register SAML Service Provider'}
					</SheetTitle>
					<SheetDescription>
						A downstream SAML 2.0 Service Provider that Authorizer (acting as
						the IdP) issues signed assertions to.
					</SheetDescription>
				</SheetHeader>

				<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
					<div className="flex items-center gap-4">
						<label className="w-32 text-sm font-medium shrink-0">Name</label>
						<Input
							placeholder="acme-app"
							value={formData.name}
							isInvalid={formData.name.trim().length === 0}
							onChange={(e) => setField('name', e.currentTarget.value)}
						/>
					</div>

					<div className="flex items-center gap-4">
						<label className="w-32 text-sm font-medium shrink-0">
							Entity ID
						</label>
						<Input
							placeholder="https://sp.example.com/saml/metadata"
							value={formData.entityId}
							isInvalid={formData.entityId.trim().length === 0}
							onChange={(e) => setField('entityId', e.currentTarget.value)}
						/>
					</div>

					<div className="flex items-center gap-4">
						<label className="w-32 text-sm font-medium shrink-0">ACS URL</label>
						<Input
							placeholder="https://sp.example.com/saml/acs"
							value={formData.acsUrl}
							isInvalid={formData.acsUrl.trim().length === 0}
							onChange={(e) => setField('acsUrl', e.currentTarget.value)}
						/>
					</div>

					<div className="flex items-start gap-4">
						<label className="w-32 text-sm font-medium shrink-0 pt-2">
							SP Certificate
						</label>
						<Textarea
							rows={5}
							className="font-mono text-xs"
							placeholder="-----BEGIN CERTIFICATE----- (PEM, optional)"
							value={formData.spCertPem}
							onChange={(e) => setField('spCertPem', e.currentTarget.value)}
						/>
					</div>

					<div className="flex items-center gap-4">
						<label className="w-32 text-sm font-medium shrink-0">
							NameID Format
						</label>
						<Input
							placeholder="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
							value={formData.nameIdFormat}
							onChange={(e) => setField('nameIdFormat', e.currentTarget.value)}
						/>
					</div>

					<div className="flex items-start gap-4">
						<label className="w-32 text-sm font-medium shrink-0 pt-2">
							Mapped Attributes
						</label>
						<Textarea
							rows={3}
							className="font-mono text-xs"
							placeholder='{"email":"email","given_name":"firstName"}'
							value={formData.mappedAttributes}
							onChange={(e) =>
								setField('mappedAttributes', e.currentTarget.value)
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
	);
};

export default UpdateSAMLServiceProviderModal;
