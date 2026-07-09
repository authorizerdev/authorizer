import React, { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { UpdateModalViews } from '../constants';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import { AddTrustedIssuer, UpdateTrustedIssuer } from '../graphql/mutation';
import type { TrustedIssuer } from '../types';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select } from './ui/select';
import { Switch } from './ui/switch';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';

const keySourceTypes = [
	'oidc_discovery',
	'static_jwks_url',
	'spiffe_bundle_endpoint',
];

const issuerTypes = ['kubernetes_sa', 'spiffe_jwt', 'oidc', 'cloud_oidc'];

interface UpdateTrustedIssuerModalProps {
	view: UpdateModalViews;
	selectedIssuer?: TrustedIssuer;
	fetchIssuers: () => void;
}

interface IssuerFormData {
	serviceAccountId: string;
	name: string;
	issuerUrl: string;
	keySourceType: string;
	jwksUrl: string;
	expectedAud: string;
	subjectClaim: string;
	allowedSubjects: string;
	issuerType: string;
	spiffeRefreshHintSeconds: string;
	isActive: boolean;
}

const initFormData: IssuerFormData = {
	serviceAccountId: '',
	name: '',
	issuerUrl: '',
	keySourceType: keySourceTypes[0],
	jwksUrl: '',
	expectedAud: '',
	subjectClaim: '',
	allowedSubjects: '',
	issuerType: issuerTypes[0],
	spiffeRefreshHintSeconds: '',
	isActive: true,
};

const UpdateTrustedIssuerModal = ({
	view,
	selectedIssuer,
	fetchIssuers,
}: UpdateTrustedIssuerModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [formData, setFormData] = useState<IssuerFormData>({ ...initFormData });

	const isEdit = view === UpdateModalViews.Edit;

	useEffect(() => {
		if (open && isEdit && selectedIssuer) {
			setFormData({
				serviceAccountId: selectedIssuer.service_account_id,
				name: selectedIssuer.name,
				issuerUrl: selectedIssuer.issuer_url,
				keySourceType: selectedIssuer.key_source_type,
				jwksUrl: selectedIssuer.jwks_url || '',
				expectedAud: selectedIssuer.expected_aud,
				subjectClaim: selectedIssuer.subject_claim,
				allowedSubjects: selectedIssuer.allowed_subjects || '',
				issuerType: selectedIssuer.issuer_type,
				spiffeRefreshHintSeconds:
					selectedIssuer.spiffe_refresh_hint_seconds != null
						? String(selectedIssuer.spiffe_refresh_hint_seconds)
						: '',
				isActive: selectedIssuer.is_active,
			});
		}
	}, [open]);

	const setField = (field: keyof IssuerFormData, value: string | boolean) =>
		setFormData({ ...formData, [field]: value });

	const validateData = () => {
		if (loading) return false;
		if (isEdit) {
			return (
				formData.name.trim().length > 0 &&
				formData.expectedAud.trim().length > 0
			);
		}
		return (
			formData.serviceAccountId.trim().length > 0 &&
			formData.name.trim().length > 0 &&
			formData.issuerUrl.trim().length > 0 &&
			formData.expectedAud.trim().length > 0
		);
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		const refreshHint = formData.spiffeRefreshHintSeconds.trim()
			? parseInt(formData.spiffeRefreshHintSeconds, 10)
			: undefined;
		let res: { error?: unknown };
		if (isEdit && selectedIssuer?.id) {
			res = await client
				.mutation(UpdateTrustedIssuer, {
					params: {
						id: selectedIssuer.id,
						name: formData.name.trim(),
						jwks_url: formData.jwksUrl.trim() || undefined,
						expected_aud: formData.expectedAud.trim(),
						allowed_subjects: formData.allowedSubjects.trim(),
						is_active: formData.isActive,
						spiffe_refresh_hint_seconds: refreshHint,
					},
				})
				.toPromise();
		} else {
			res = await client
				.mutation(AddTrustedIssuer, {
					params: {
						service_account_id: formData.serviceAccountId.trim(),
						name: formData.name.trim(),
						issuer_url: formData.issuerUrl.trim(),
						key_source_type: formData.keySourceType,
						jwks_url: formData.jwksUrl.trim() || undefined,
						expected_aud: formData.expectedAud.trim(),
						subject_claim: formData.subjectClaim.trim() || undefined,
						allowed_subjects: formData.allowedSubjects.trim() || undefined,
						issuer_type: formData.issuerType,
						spiffe_refresh_hint_seconds: refreshHint,
					},
				})
				.toPromise();
		}
		setLoading(false);
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to save trusted issuer'),
				),
			);
			return;
		}
		toast.success(isEdit ? 'Trusted issuer updated' : 'Trusted issuer added');
		setFormData({ ...initFormData });
		setOpen(false);
		fetchIssuers();
	};

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button size="sm" onClick={() => setOpen(true)}>
					<Plus className="mr-2 h-4 w-4" />
					Add Trusted Issuer
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
							{isEdit ? 'Edit Trusted Issuer' : 'Add Trusted Issuer'}
						</SheetTitle>
						<SheetDescription>
							External JWT issuers bound to a service account for workload
							authentication.
						</SheetDescription>
					</SheetHeader>

					<div className="mt-6 space-y-5 rounded-md border border-gray-200 p-5">
						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								Service Account ID
							</label>
							<Input
								placeholder="service account (client) id"
								value={formData.serviceAccountId}
								isInvalid={
									!isEdit && formData.serviceAccountId.trim().length === 0
								}
								disabled={isEdit}
								onChange={(e) =>
									setField('serviceAccountId', e.currentTarget.value)
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">Name</label>
							<Input
								placeholder="prod-cluster"
								value={formData.name}
								isInvalid={formData.name.trim().length === 0}
								onChange={(e) => setField('name', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								Issuer URL
							</label>
							<Input
								placeholder="https://kubernetes.default.svc"
								value={formData.issuerUrl}
								isInvalid={!isEdit && formData.issuerUrl.trim().length === 0}
								disabled={isEdit}
								onChange={(e) => setField('issuerUrl', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								Issuer Type
							</label>
							<Select
								value={formData.issuerType}
								disabled={isEdit}
								onChange={(e) => setField('issuerType', e.currentTarget.value)}
							>
								{issuerTypes.map((type) => (
									<option key={type} value={type}>
										{type}
									</option>
								))}
							</Select>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								Key Source
							</label>
							<Select
								value={formData.keySourceType}
								disabled={isEdit}
								onChange={(e) =>
									setField('keySourceType', e.currentTarget.value)
								}
							>
								{keySourceTypes.map((type) => (
									<option key={type} value={type}>
										{type}
									</option>
								))}
							</Select>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								JWKS URL
							</label>
							<Input
								placeholder="https://issuer.example.com/jwks (optional)"
								value={formData.jwksUrl}
								onChange={(e) => setField('jwksUrl', e.currentTarget.value)}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								Expected Audience
							</label>
							<Input
								placeholder="https://authorizer.example.com"
								value={formData.expectedAud}
								isInvalid={formData.expectedAud.trim().length === 0}
								onChange={(e) => setField('expectedAud', e.currentTarget.value)}
							/>
						</div>

						{!isEdit && (
							<div className="flex items-center gap-4">
								<label className="w-40 text-sm font-medium shrink-0">
									Subject Claim
								</label>
								<Input
									placeholder="sub (default)"
									value={formData.subjectClaim}
									onChange={(e) =>
										setField('subjectClaim', e.currentTarget.value)
									}
								/>
							</div>
						)}

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								Allowed Subjects
							</label>
							<Input
								placeholder="comma-separated subjects; empty = deny all"
								value={formData.allowedSubjects}
								onChange={(e) =>
									setField('allowedSubjects', e.currentTarget.value)
								}
							/>
						</div>

						<div className="flex items-center gap-4">
							<label className="w-40 text-sm font-medium shrink-0">
								SPIFFE Refresh Hint (s)
							</label>
							<Input
								type="number"
								placeholder="optional"
								value={formData.spiffeRefreshHintSeconds}
								onChange={(e) =>
									setField('spiffeRefreshHintSeconds', e.currentTarget.value)
								}
							/>
						</div>

						{isEdit && (
							<div className="flex items-center gap-4">
								<label className="w-40 text-sm font-medium shrink-0">
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

export default UpdateTrustedIssuerModal;
