import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { Plus, RefreshCw, Upload } from 'lucide-react';
import dayjs from 'dayjs';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';
import {
	ListSAMLServiceProvidersQuery,
	ListSAMLIDPKeysQuery,
} from '../graphql/queries';
import {
	DeleteSAMLServiceProvider,
	RotateSAMLIDPCert,
	RetireSAMLIDPKey,
	ImportSAMLSPMetadata,
} from '../graphql/mutation';
import type {
	SAMLServiceProvider,
	SAMLIDPKey,
	SAMLSPMetadataParseResult,
} from '../types';
import UpdateSAMLServiceProviderModal from './UpdateSAMLServiceProviderModal';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Textarea } from './ui/textarea';
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from './ui/card';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from './ui/dialog';
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
	SheetDescription,
	SheetFooter,
} from './ui/sheet';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from './ui/table';

interface SAMLServiceProvidersProps {
	orgId: string;
	// orgSlug is the URL-safe org name used in the IdP metadata URL.
	orgSlug: string;
}

const keyStatusVariant = (status: string) => {
	if (status === 'current') return 'success';
	if (status === 'active') return 'default';
	return 'secondary';
};

const SAMLServiceProviders = ({
	orgId,
	orgSlug,
}: SAMLServiceProvidersProps) => {
	const client = useClient();

	const [providers, setProviders] = useState<SAMLServiceProvider[]>([]);
	const [keys, setKeys] = useState<SAMLIDPKey[]>([]);

	// Create/edit modal.
	const [modalOpen, setModalOpen] = useState(false);
	const [editingProvider, setEditingProvider] =
		useState<SAMLServiceProvider | null>(null);
	const [prefill, setPrefill] = useState<SAMLSPMetadataParseResult | null>(
		null,
	);

	// Import SP metadata sheet.
	const [importOpen, setImportOpen] = useState(false);
	const [metadataXml, setMetadataXml] = useState('');
	const [importing, setImporting] = useState(false);

	// Confirm dialogs.
	const [providerToDelete, setProviderToDelete] =
		useState<SAMLServiceProvider | null>(null);
	const [keyToRetire, setKeyToRetire] = useState<SAMLIDPKey | null>(null);
	const [rotateOpen, setRotateOpen] = useState(false);

	const fetchProviders = async () => {
		if (!orgId) return;
		const res = await client
			.query<{
				_list_saml_service_providers: {
					saml_service_providers: SAMLServiceProvider[];
				};
			}>(ListSAMLServiceProvidersQuery, { params: { org_id: orgId } })
			.toPromise();
		setProviders(
			res.data?._list_saml_service_providers?.saml_service_providers || [],
		);
	};

	const fetchKeys = async () => {
		if (!orgId) return;
		const res = await client
			.query<{ _list_saml_idp_keys: SAMLIDPKey[] }>(ListSAMLIDPKeysQuery, {
				params: { org_id: orgId },
			})
			.toPromise();
		setKeys(res.data?._list_saml_idp_keys || []);
	};

	useEffect(() => {
		fetchProviders();
		fetchKeys();
	}, [orgId]);

	const openCreate = () => {
		setEditingProvider(null);
		setPrefill(null);
		setModalOpen(true);
	};

	const openEdit = (provider: SAMLServiceProvider) => {
		setEditingProvider(provider);
		setPrefill(null);
		setModalOpen(true);
	};

	const importMetadata = async () => {
		if (!metadataXml.trim()) return;
		setImporting(true);
		const res = await client
			.mutation<{
				_import_saml_sp_metadata: SAMLSPMetadataParseResult;
			}>(ImportSAMLSPMetadata, { params: { metadata_xml: metadataXml.trim() } })
			.toPromise();
		setImporting(false);
		if (res.error || !res.data?._import_saml_sp_metadata) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to parse SP metadata'),
				),
			);
			return;
		}
		// Prefill the create form with the parsed values.
		setEditingProvider(null);
		setPrefill(res.data._import_saml_sp_metadata);
		setImportOpen(false);
		setMetadataXml('');
		setModalOpen(true);
	};

	const deleteProviderHandler = async () => {
		if (!providerToDelete) return;
		const res = await client
			.mutation(DeleteSAMLServiceProvider, {
				params: { id: providerToDelete.id },
			})
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(
						res.error,
						'Failed to delete SAML service provider',
					),
				),
			);
			return;
		}
		toast.success('SAML service provider deleted');
		setProviderToDelete(null);
		fetchProviders();
	};

	const rotateCertHandler = async () => {
		const res = await client
			.mutation(RotateSAMLIDPCert, { params: { org_id: orgId } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(
						res.error,
						'Failed to rotate signing certificate',
					),
				),
			);
			return;
		}
		toast.success('Signing certificate rotated');
		setRotateOpen(false);
		fetchKeys();
	};

	const retireKeyHandler = async () => {
		if (!keyToRetire) return;
		const res = await client
			.mutation(RetireSAMLIDPKey, { params: { id: keyToRetire.id } })
			.toPromise();
		if (res.error) {
			toast.error(
				capitalizeFirstLetter(
					getGraphQLErrorMessage(res.error, 'Failed to retire signing key'),
				),
			);
			return;
		}
		toast.success('Signing key retired');
		setKeyToRetire(null);
		fetchKeys();
	};

	const metadataUrl = `${window.location.origin}/saml/idp/${orgSlug}/metadata`;

	return (
		<>
			{/* SAML IdP: registered service providers */}
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>SAML IdP — Service Providers</CardTitle>
							<CardDescription>
								Downstream SAML 2.0 Service Providers this organization issues
								signed assertions to. Authorizer acts as the Identity Provider.
							</CardDescription>
						</div>
						<div className="flex gap-2">
							<Button
								variant="outline"
								size="sm"
								onClick={() => setImportOpen(true)}
							>
								<Upload className="mr-2 h-4 w-4" />
								Import SP Metadata
							</Button>
							<Button size="sm" onClick={openCreate}>
								<Plus className="mr-2 h-4 w-4" />
								Register Service Provider
							</Button>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					{providers.length > 0 ? (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Name</TableHead>
									<TableHead>Entity ID</TableHead>
									<TableHead>ACS URL</TableHead>
									<TableHead>IdP Initiated</TableHead>
									<TableHead>Active</TableHead>
									<TableHead className="text-right">Actions</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{providers.map((sp) => (
									<TableRow key={sp.id}>
										<TableCell>{sp.name}</TableCell>
										<TableCell className="font-mono text-xs">
											{sp.entity_id}
										</TableCell>
										<TableCell className="font-mono text-xs">
											{sp.acs_url}
										</TableCell>
										<TableCell>
											{sp.allow_idp_initiated ? 'Allowed' : 'Disabled'}
										</TableCell>
										<TableCell>
											<Badge variant={sp.is_active ? 'success' : 'warning'}>
												{sp.is_active.toString()}
											</Badge>
										</TableCell>
										<TableCell className="text-right">
											<div className="flex justify-end gap-2">
												<Button
													variant="outline"
													size="sm"
													onClick={() => openEdit(sp)}
												>
													Edit
												</Button>
												<Button
													variant="destructive"
													size="sm"
													onClick={() => setProviderToDelete(sp)}
												>
													Delete
												</Button>
											</div>
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					) : (
						<p className="text-sm text-gray-400">
							No SAML service providers registered.
						</p>
					)}
				</CardContent>
			</Card>

			{/* SAML IdP: signing keys */}
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>SAML IdP — Signing Keys</CardTitle>
							<CardDescription>
								Signing keypairs published in this organization's IdP metadata.
								Rotating generates a new "current" key; the previous one stays
								"active" until retired.
							</CardDescription>
						</div>
						<Button
							variant="outline"
							size="sm"
							onClick={() => setRotateOpen(true)}
						>
							<RefreshCw className="mr-2 h-4 w-4" />
							Rotate Certificate
						</Button>
					</div>
				</CardHeader>
				<CardContent>
					<p className="mb-4 text-sm text-gray-500">
						IdP metadata URL:{' '}
						<span className="font-mono text-xs text-gray-700">
							{metadataUrl}
						</span>
					</p>
					{keys.length > 0 ? (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Status</TableHead>
									<TableHead>Algorithm</TableHead>
									<TableHead>Created</TableHead>
									<TableHead className="text-right">Actions</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{keys.map((key) => (
									<TableRow key={key.id}>
										<TableCell>
											<Badge variant={keyStatusVariant(key.status)}>
												{key.status}
											</Badge>
										</TableCell>
										<TableCell className="font-mono text-xs">
											{key.algorithm}
										</TableCell>
										<TableCell>
											{key.created_at
												? dayjs.unix(key.created_at).format('MMM D, YYYY')
												: '—'}
										</TableCell>
										<TableCell className="text-right">
											{key.status !== 'current' && key.status !== 'retired' ? (
												<Button
													variant="outline"
													size="sm"
													onClick={() => setKeyToRetire(key)}
												>
													Retire
												</Button>
											) : null}
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					) : (
						<p className="text-sm text-gray-400">No signing keys yet.</p>
					)}
				</CardContent>
			</Card>

			<UpdateSAMLServiceProviderModal
				open={modalOpen}
				onOpenChange={setModalOpen}
				orgId={orgId}
				selectedProvider={editingProvider}
				prefill={prefill}
				onSaved={fetchProviders}
			/>

			{/* Import SP metadata */}
			<Sheet open={importOpen} onOpenChange={setImportOpen}>
				<SheetContent className="overflow-y-auto sm:max-w-2xl">
					<SheetHeader>
						<SheetTitle>Import SP Metadata</SheetTitle>
						<SheetDescription>
							Paste the Service Provider's SAML metadata XML. Parsed values
							prefill the registration form — no record is created until you
							save.
						</SheetDescription>
					</SheetHeader>
					<div className="mt-6">
						<Textarea
							rows={16}
							className="font-mono text-xs"
							placeholder="<md:EntityDescriptor ...>"
							value={metadataXml}
							onChange={(e) => setMetadataXml(e.currentTarget.value)}
						/>
					</div>
					<SheetFooter className="mt-6">
						<Button
							onClick={importMetadata}
							isLoading={importing}
							disabled={importing || metadataXml.trim().length === 0}
						>
							Parse & Continue
						</Button>
					</SheetFooter>
				</SheetContent>
			</Sheet>

			{/* Confirm: delete provider */}
			<Dialog
				open={!!providerToDelete}
				onOpenChange={(isOpen) => {
					if (!isOpen) setProviderToDelete(null);
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete SAML Service Provider</DialogTitle>
						<DialogDescription>Are you sure?</DialogDescription>
					</DialogHeader>
					<div className="rounded-md border border-red-300 bg-red-50 p-4">
						<p className="text-sm">
							Service provider <strong>{providerToDelete?.name}</strong> will be
							deleted permanently! SSO into it will stop working.
						</p>
					</div>
					<DialogFooter>
						<Button variant="destructive" onClick={deleteProviderHandler}>
							Delete
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Confirm: rotate certificate */}
			<Dialog open={rotateOpen} onOpenChange={setRotateOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Rotate Signing Certificate</DialogTitle>
						<DialogDescription>Are you sure?</DialogDescription>
					</DialogHeader>
					<div className="rounded-md border border-yellow-300 bg-yellow-50 p-4">
						<p className="text-sm">
							A new signing keypair becomes "current". The previous key stays
							published as "active" until you retire it.
						</p>
					</div>
					<DialogFooter>
						<Button onClick={rotateCertHandler}>
							<RefreshCw className="mr-2 h-4 w-4" />
							Rotate
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Confirm: retire key */}
			<Dialog
				open={!!keyToRetire}
				onOpenChange={(isOpen) => {
					if (!isOpen) setKeyToRetire(null);
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Retire Signing Key</DialogTitle>
						<DialogDescription>Are you sure?</DialogDescription>
					</DialogHeader>
					<div className="rounded-md border border-yellow-300 bg-yellow-50 p-4">
						<p className="text-sm">
							This key will be dropped from the published IdP metadata. SPs
							still pinning it will fail to verify assertions.
						</p>
					</div>
					<DialogFooter>
						<Button variant="destructive" onClick={retireKeyHandler}>
							Retire
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
};

export default SAMLServiceProviders;
