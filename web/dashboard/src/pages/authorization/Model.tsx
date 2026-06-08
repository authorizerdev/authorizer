import React, { useCallback, useEffect, useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { AlertCircle, Save, LayoutGrid, Code2, Info } from 'lucide-react';
import { FgaGetModelQuery, AdminRolesQuery } from '../../graphql/queries';
import { FgaWriteModel } from '../../graphql/mutation';
import { Button } from '../../components/ui/button';
import { Textarea } from '../../components/ui/textarea';
import { Skeleton } from '../../components/ui/skeleton';
import { Badge } from '../../components/ui/badge';
import FgaNotEnabled from '../../components/FgaNotEnabled';
import ModelBuilder from './ModelBuilder';
import AuthSteps, { Example, NextStep } from './AuthSteps';
import {
	generateDsl,
	parseDsl,
	validateModel,
	summarize,
	rolesTemplate,
	TEMPLATES,
	type ModelDraft,
} from './modelDsl';
import { isFgaNotEnabledError } from '../../lib/utils';
import type {
	FgaGetModelResponse,
	FgaWriteModelResponse,
	AdminRolesResponse,
} from '../../types';

type Mode = 'builder' | 'dsl';

const STARTER: ModelDraft = { types: [{ name: 'user', relations: [] }] };

const TabButton = ({
	active,
	onClick,
	icon,
	children,
}: {
	active: boolean;
	onClick: () => void;
	icon: React.ReactNode;
	children: React.ReactNode;
}) => (
	<button
		type="button"
		onClick={onClick}
		className={`inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
			active ? 'bg-blue-50 text-blue-600' : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700'
		}`}
	>
		{icon}
		{children}
	</button>
);

const Model = () => {
	const client = useClient();
	const [loading, setLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [fgaDisabled, setFgaDisabled] = useState(false);
	const [mode, setMode] = useState<Mode>('builder');
	const [builderModel, setBuilderModel] = useState<ModelDraft>(STARTER);
	const [dsl, setDsl] = useState('');
	const [modelId, setModelId] = useState('');
	const [validationError, setValidationError] = useState('');
	const [roles, setRoles] = useState<string[]>([]);

	// Fetch the instance's configured roles (admin _env) so the builder can offer
	// a template built from the real roles. Best-effort: ignore failures.
	useEffect(() => {
		client
			.query<AdminRolesResponse>(AdminRolesQuery, {})
			.toPromise()
			.then((res) => {
				const r = res.data?._env?.ROLES;
				if (Array.isArray(r)) setRoles(r.filter(Boolean));
			})
			.catch(() => {
				/* roles template just won't be offered */
			});
	}, [client]);

	const fetchModel = useCallback(async () => {
		setLoading(true);
		try {
			const res = await client
				.query<FgaGetModelResponse>(FgaGetModelQuery, {}, { requestPolicy: 'network-only' })
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) setFgaDisabled(true);
				else toast.error('Failed to load authorization model');
				return;
			}

			const current = res.data?._fga_get_model;
			if (current?.dsl) {
				setDsl(current.dsl);
				setModelId(current.id || '');
				const parsed = parseDsl(current.dsl);
				if (parsed.supported && parsed.model) {
					setBuilderModel(parsed.model);
					setMode('builder');
				} else {
					// Model uses constructs the builder can't represent — edit in DSL.
					setMode('dsl');
				}
			} else {
				// No model yet — start in the builder with a base `user` type.
				setBuilderModel(STARTER);
				setMode('builder');
			}
		} catch {
			toast.error('Failed to load authorization model');
		} finally {
			setLoading(false);
		}
	}, [client]);

	useEffect(() => {
		fetchModel();
	}, [fetchModel]);

	const switchToDsl = () => {
		setDsl(generateDsl(builderModel));
		setValidationError('');
		setMode('dsl');
	};

	const switchToBuilder = () => {
		const parsed = parseDsl(dsl);
		if (parsed.supported && parsed.model) {
			setBuilderModel(parsed.model);
			setValidationError('');
			setMode('builder');
		} else {
			toast.error('This model uses advanced constructs — keep editing in DSL');
		}
	};

	const applyTemplate = (model: ModelDraft) => {
		setBuilderModel(model);
		setMode('builder');
		setValidationError('');
	};

	const handleSave = async () => {
		let dslToSave = dsl;
		if (mode === 'builder') {
			const err = validateModel(builderModel);
			if (err) {
				setValidationError(err);
				return;
			}
			dslToSave = generateDsl(builderModel);
		} else if (!dsl.trim()) {
			setValidationError('Model DSL cannot be empty.');
			return;
		}
		setValidationError('');
		setSaving(true);
		try {
			const res = await client
				.mutation<FgaWriteModelResponse>(FgaWriteModel, { params: { dsl: dslToSave } })
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) setFgaDisabled(true);
				else {
					setValidationError(res.error.message.replace('[GraphQL] ', ''));
					toast.error('Failed to save authorization model');
				}
				return;
			}

			if (res.data?._fga_write_model) {
				setModelId(res.data._fga_write_model.id);
				setDsl(res.data._fga_write_model.dsl || dslToSave);
				toast.success('Authorization model saved');
			}
		} catch {
			toast.error('Failed to save authorization model');
		} finally {
			setSaving(false);
		}
	};

	const summary = mode === 'builder' ? summarize(builderModel) : [];

	// Offer a template built from the instance's configured roles first.
	const roleModel = rolesTemplate(roles);
	const templates = [
		...(roleModel
			? [
					{
						name: 'RBAC — your roles',
						description: `Your configured roles (${roles.join(', ')}) as relations on a resource.`,
						model: roleModel,
					},
			  ]
			: []),
		...TEMPLATES,
	];

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<AuthSteps current={1} />
			<div className="my-4 flex items-start justify-between gap-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">Step 1 · Define the model</h1>
					<p className="mt-1 max-w-2xl text-sm text-gray-500">
						The model is your permission <strong>rulebook</strong>: the object{' '}
						<strong>types</strong> you protect, the <strong>relations</strong> on them (owner,
						editor, viewer…), and how permissions are computed. You write it once.
					</p>
				</div>
				{!fgaDisabled && !loading && (
					<Button onClick={handleSave} disabled={saving}>
						<Save className="mr-2 h-4 w-4" />
						{saving ? 'Saving…' : 'Save model'}
					</Button>
				)}
			</div>
			{!fgaDisabled && !loading && (
				<div className="mb-4">
					<Example>
						<strong>Example:</strong> a <code className="rounded bg-white px-1 py-0.5 text-xs">document</code> has{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">viewer</code> and{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">editor</code> relations, and{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">can_view = viewer or editor</code>. Pick a
						template below to start, then customize.
					</Example>
				</div>
			)}

			{loading ? (
				<div className="space-y-3">
					<Skeleton className="h-9 w-64" />
					<Skeleton className="h-64 w-full" />
				</div>
			) : fgaDisabled ? (
				<FgaNotEnabled />
			) : (
				<div className="space-y-4">
					<div className="flex flex-wrap items-center justify-between gap-3">
						<div className="inline-flex items-center gap-1 rounded-lg bg-gray-50 p-1">
							<TabButton
								active={mode === 'builder'}
								onClick={() => mode !== 'builder' && switchToBuilder()}
								icon={<LayoutGrid className="h-4 w-4" />}
							>
								Builder
							</TabButton>
							<TabButton
								active={mode === 'dsl'}
								onClick={() => mode !== 'dsl' && switchToDsl()}
								icon={<Code2 className="h-4 w-4" />}
							>
								DSL (advanced)
							</TabButton>
						</div>

						<div className="flex items-center gap-3">
							{modelId && (
								<span className="flex items-center gap-1.5 text-xs text-gray-500">
									active model
									<Badge variant="secondary">{modelId.slice(0, 12)}…</Badge>
								</span>
							)}
						</div>
					</div>

					{mode === 'builder' && (
						<div className="flex flex-wrap items-center gap-2 rounded-lg border border-dashed border-gray-200 p-2">
							<span className="px-1 text-xs font-medium text-gray-500">Start from a template:</span>
							{templates.map((tpl) => (
								<button
									key={tpl.name}
									type="button"
									onClick={() => applyTemplate(tpl.model)}
									title={tpl.description}
									className="rounded-md border border-gray-200 bg-white px-2.5 py-1 text-xs font-medium text-gray-700 transition-colors hover:border-blue-300 hover:bg-blue-50 hover:text-blue-700"
								>
									{tpl.name}
								</button>
							))}
						</div>
					)}

					{mode === 'builder' ? (
						<div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
							<div className="lg:col-span-2">
								<ModelBuilder model={builderModel} onChange={setBuilderModel} />
							</div>
							<aside className="lg:col-span-1">
								<div className="sticky top-4 rounded-xl border border-gray-100 bg-gray-50 p-4">
									<div className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-gray-500">
										<Info className="h-3.5 w-3.5" />
										What this model means
									</div>
									{summary.length ? (
										<ul className="space-y-1.5 text-xs leading-relaxed text-gray-600">
											{summary.map((s, i) => (
												<li key={i}>• {s}</li>
											))}
										</ul>
									) : (
										<p className="text-xs text-gray-400">
											Add a type with a relation to see a plain-English summary.
										</p>
									)}
								</div>
							</aside>
						</div>
					) : (
						<Textarea
							value={dsl}
							onChange={(e) => setDsl(e.target.value)}
							spellCheck={false}
							className="min-h-[420px] font-mono text-xs leading-relaxed"
							placeholder={
								'model\n  schema 1.1\n\ntype user\n\ntype document\n  relations\n    define viewer: [user]\n    define editor: [user]\n    define can_view: viewer or editor'
							}
						/>
					)}

					{validationError && (
						<div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
							<AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
							<span className="whitespace-pre-wrap break-words">{validationError}</span>
						</div>
					)}

					<div className="flex items-center justify-between border-t border-gray-100 pt-4 text-sm text-gray-500">
						<span>Save the model, then grant access with relationship tuples.</span>
						<NextStep to="/authorization/tuples" label="Next: grant access" />
					</div>
				</div>
			)}
		</div>
	);
};

export default Model;
