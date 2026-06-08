import React, { useCallback, useEffect, useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { AlertCircle, Save, Info } from 'lucide-react';
import { FgaGetModelQuery, AdminRolesQuery } from '../../graphql/queries';
import { FgaWriteModel } from '../../graphql/mutation';
import { Button } from '../../components/ui/button';
import { Textarea } from '../../components/ui/textarea';
import { Skeleton } from '../../components/ui/skeleton';
import { Badge } from '../../components/ui/badge';
import FgaNotEnabled from '../../components/FgaNotEnabled';
import AuthSteps, { Example, NextStep } from './AuthSteps';
import DocsLinks from './DocsLinks';
import { generateDsl, parseDsl, summarize, rolesTemplate, MODEL_EXAMPLES } from './modelDsl';
import { isFgaNotEnabledError } from '../../lib/utils';
import type {
	FgaGetModelResponse,
	FgaWriteModelResponse,
	AdminRolesResponse,
} from '../../types';

const PLACEHOLDER = `model
  schema 1.1

type user

type document
  relations
    define viewer: [user]
    define editor: [user]
    define can_view: viewer or editor`;

const Model = () => {
	const client = useClient();
	const [loading, setLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [fgaDisabled, setFgaDisabled] = useState(false);
	const [dsl, setDsl] = useState('');
	const [modelId, setModelId] = useState('');
	const [validationError, setValidationError] = useState('');
	const [roles, setRoles] = useState<string[]>([]);

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

	// Configured roles → a "your roles" example.
	useEffect(() => {
		client
			.query<AdminRolesResponse>(AdminRolesQuery, {})
			.toPromise()
			.then((res) => {
				const r = res.data?._env?.ROLES;
				if (Array.isArray(r)) setRoles(r.filter(Boolean));
			})
			.catch(() => {});
	}, [client]);

	const applyExample = (name: string, exampleDsl: string) => {
		if (
			dsl.trim() &&
			dsl.trim() !== exampleDsl.trim() &&
			!window.confirm(
				`Replace the editor with the "${name}" example? Unsaved changes will be lost.`,
			)
		) {
			return;
		}
		setDsl(exampleDsl);
		setValidationError('');
		toast.success(`Loaded the "${name}" example`);
	};

	const handleSave = async () => {
		if (!dsl.trim()) {
			setValidationError('The model cannot be empty. Pick an example to start.');
			return;
		}
		setValidationError('');
		setSaving(true);
		try {
			const res = await client
				.mutation<FgaWriteModelResponse>(FgaWriteModel, { params: { dsl } })
				.toPromise();
			if (res.error) {
				if (isFgaNotEnabledError(res.error)) setFgaDisabled(true);
				else {
					setValidationError(res.error.message.replace('[GraphQL] ', ''));
					toast.error('Could not save — check the model syntax');
				}
				return;
			}
			if (res.data?._fga_write_model) {
				setModelId(res.data._fga_write_model.id);
				setDsl(res.data._fga_write_model.dsl || dsl);
				toast.success('Authorization model saved');
			}
		} catch {
			toast.error('Failed to save authorization model');
		} finally {
			setSaving(false);
		}
	};

	const roleModel = rolesTemplate(roles);
	const examples = [
		...(roleModel
			? [
					{
						name: 'Your roles',
						description: `Your configured roles (${roles.join(', ')}) as a starting point.`,
						dsl: generateDsl(roleModel).trimEnd(),
					},
			  ]
			: []),
		...MODEL_EXAMPLES,
	];

	const parsed = parseDsl(dsl);
	const summary = parsed.supported && parsed.model ? summarize(parsed.model) : [];

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<AuthSteps current={1} />

			<div className="my-4 flex items-start justify-between gap-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">Step 1 · Define the model</h1>
					<p className="mt-1 max-w-2xl text-sm text-gray-500">
						The model is your permission <strong>rulebook</strong>: the object{' '}
						<strong>types</strong> you protect (document, folder…), their{' '}
						<strong>relations</strong> (owner, editor, viewer…), and how permissions are
						computed. You write it once.
					</p>
				</div>
				{!fgaDisabled && !loading && (
					<Button onClick={handleSave} disabled={saving}>
						<Save className="mr-2 h-4 w-4" />
						{saving ? 'Saving…' : 'Save model'}
					</Button>
				)}
			</div>

			{loading ? (
				<div className="space-y-3">
					<Skeleton className="h-9 w-64" />
					<Skeleton className="h-72 w-full" />
				</div>
			) : fgaDisabled ? (
				<FgaNotEnabled />
			) : (
				<div className="space-y-5">
					<Example>
						<strong>Example:</strong> a{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">document</code> has{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">viewer</code> and{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">editor</code> relations, and{' '}
						<code className="rounded bg-white px-1 py-0.5 text-xs">can_view = viewer or editor</code>.
						Start from an example below and edit it.
					</Example>

					{/* Example templates with descriptions */}
					<div>
						<p className="mb-2 text-sm font-medium text-gray-700">Start from an example</p>
						<div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
							{examples.map((ex) => (
								<button
									key={ex.name}
									type="button"
									onClick={() => applyExample(ex.name, ex.dsl)}
									className="rounded-xl border border-gray-200 bg-white p-3 text-left transition-colors hover:border-blue-300 hover:bg-blue-50"
								>
									<span className="block text-sm font-medium text-gray-800">{ex.name}</span>
									<span className="mt-0.5 block text-xs leading-relaxed text-gray-500">
										{ex.description}
									</span>
								</button>
							))}
						</div>
					</div>

					{/* The model */}
					<div>
						<div className="mb-1.5 flex items-center justify-between">
							<label htmlFor="model-dsl" className="text-sm font-medium text-gray-700">
								Model
							</label>
							{modelId && (
								<span className="flex items-center gap-1.5 text-xs text-gray-400">
									active version
									<Badge variant="secondary">{modelId.slice(0, 12)}…</Badge>
								</span>
							)}
						</div>
						<div className="mb-2 flex items-start gap-2 rounded-lg border border-gray-100 bg-gray-50 p-3 text-xs leading-relaxed text-gray-600">
							<Info className="mt-0.5 h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
							<div>
								<strong className="text-gray-700">About model versions.</strong> There is always
								exactly one <em>active</em> model. Saving creates a new <strong>immutable
								version</strong> and makes it active; earlier versions are retained so requests
								already in flight stay valid. OpenFGA models are <strong>append-only</strong> — an
								individual version cannot be deleted. To change the rules, save a new version; to
								remove everything, reset the store (deletes the model and all tuples). Separate
								models require separate stores, which aren&rsquo;t exposed here.
							</div>
						</div>
						<Textarea
							id="model-dsl"
							value={dsl}
							onChange={(e) => setDsl(e.target.value)}
							spellCheck={false}
							className="min-h-[360px] font-mono text-xs leading-relaxed"
							placeholder={PLACEHOLDER}
						/>
					</div>

					{summary.length > 0 && (
						<div className="rounded-lg border border-gray-100 bg-gray-50 p-3">
							<p className="mb-1.5 text-xs font-medium uppercase tracking-wide text-gray-500">
								In plain English
							</p>
							<ul className="space-y-1 text-xs leading-relaxed text-gray-600">
								{summary.map((s, i) => (
									<li key={i}>• {s}</li>
								))}
							</ul>
						</div>
					)}

					{validationError && (
						<div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
							<AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
							<span className="whitespace-pre-wrap break-words">{validationError}</span>
						</div>
					)}

					<DocsLinks />

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
