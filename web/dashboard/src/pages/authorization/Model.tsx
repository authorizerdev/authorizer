import React, { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useClient } from 'urql';
import { toast } from 'sonner';
import {
	AlertCircle,
	Save,
	Info,
	RotateCcw,
	AlertTriangle,
	LayoutTemplate,
} from 'lucide-react';
import { FgaGetModelQuery, FgaReadTuplesQuery } from '../../graphql/queries';
import { FgaWriteModel, FgaReset } from '../../graphql/mutation';
import { Button } from '../../components/ui/button';
import { Textarea } from '../../components/ui/textarea';
import { Skeleton } from '../../components/ui/skeleton';
import { Badge } from '../../components/ui/badge';
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
	DialogFooter,
	DialogClose,
} from '../../components/ui/dialog';
import { Input } from '../../components/ui/input';
import FgaNotEnabled from '../../components/FgaNotEnabled';
import AuthSteps, { Example, NextStep } from './AuthSteps';
import DocsLinks from './DocsLinks';
import RbacBuilder from './RbacBuilder';
import { parseDsl, summarize, MODEL_EXAMPLES } from './modelDsl';
import { isFgaNotEnabledError } from '../../lib/utils';
import type {
	FgaGetModelResponse,
	FgaWriteModelResponse,
	FgaReadTuplesResponse,
	FgaResetResponse,
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
	// Step 1 has two ways in: a friendly roles × permissions matrix ("simple",
	// the default for newcomers) and the raw OpenFGA DSL ("advanced"). An admin
	// who already has a saved model lands in advanced so they see their model.
	const [mode, setMode] = useState<'simple' | 'advanced'>('simple');
	// Reset is destructive and guarded: it is only allowed when no relationship
	// tuples exist (the backend enforces this too). tuplesExist gates the action.
	const [tuplesExist, setTuplesExist] = useState(false);
	const [resetOpen, setResetOpen] = useState(false);
	const [resetting, setResetting] = useState(false);
	const [confirmText, setConfirmText] = useState('');
	// The example catalog lives in a modal so the editor stays the focus.
	const [exampleOpen, setExampleOpen] = useState(false);

	const checkTuples = useCallback(async () => {
		try {
			const res = await client
				.query<FgaReadTuplesResponse>(
					FgaReadTuplesQuery,
					{ params: { page_size: 1 } },
					{ requestPolicy: 'network-only' },
				)
				.toPromise();
			setTuplesExist((res.data?._fga_read_tuples?.tuples?.length ?? 0) > 0);
		} catch {
			// Best-effort; leave the gate closed-safe (assume tuples may exist).
		}
	}, [client]);

	const fetchModel = useCallback(async () => {
		setLoading(true);
		try {
			const res = await client
				.query<FgaGetModelResponse>(
					FgaGetModelQuery,
					{},
					{ requestPolicy: 'network-only' },
				)
				.toPromise();
			if (res.error) {
				if (isFgaNotEnabledError(res.error)) setFgaDisabled(true);
				else if (/no authorization model/i.test(res.error.message)) {
					// A store with no model yet is the normal starting state, not an
					// error — leave the builder visible. (Newer servers already
					// return an empty model; this guards older ones.)
				} else toast.error('Failed to load authorization model');
				return;
			}
			const current = res.data?._fga_get_model;
			if (current?.dsl) {
				setDsl(current.dsl);
				setModelId(current.id || '');
				// Show an existing model in the editor rather than the builder.
				setMode('advanced');
			}
		} catch {
			toast.error('Failed to load authorization model');
		} finally {
			setLoading(false);
		}
	}, [client]);

	useEffect(() => {
		fetchModel();
		checkTuples();
	}, [fetchModel, checkTuples]);

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
		const toSave = dsl.trim();
		if (!toSave) {
			setValidationError(
				'The model cannot be empty. Pick an example to start.',
			);
			return;
		}
		setValidationError('');
		setSaving(true);
		try {
			const res = await client
				.mutation<FgaWriteModelResponse>(FgaWriteModel, {
					params: { dsl: toSave },
				})
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

	const handleReset = async () => {
		setResetting(true);
		try {
			const res = await client
				.mutation<FgaResetResponse>(FgaReset, {})
				.toPromise();
			if (res.error) {
				toast.error(res.error.message.replace('[GraphQL] ', ''));
				return;
			}
			// Store wiped: clear the editor and local state, then re-check.
			setDsl('');
			setModelId('');
			setValidationError('');
			setResetOpen(false);
			setConfirmText('');
			await checkTuples();
			toast.success('Authorization model reset — start fresh from an example');
		} catch {
			toast.error('Failed to reset authorization model');
		} finally {
			setResetting(false);
		}
	};

	const examples = MODEL_EXAMPLES;

	const parsed = parseDsl(dsl);
	const summary =
		parsed.supported && parsed.model ? summarize(parsed.model) : [];

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<AuthSteps current={1} />

			<div className="my-4 flex items-start justify-between gap-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">
						Step 1 · Define the model
					</h1>
					<p className="mt-1 max-w-2xl text-sm text-gray-500">
						The model is your permission <strong>rulebook</strong>: the object{' '}
						<strong>types</strong> you protect (document, folder…), their{' '}
						<strong>relations</strong> (owner, editor, viewer…), and how
						permissions are computed. You write it once.
					</p>
				</div>
				{!fgaDisabled && !loading && mode === 'advanced' && (
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
					{/* Two ways into Step 1: a friendly matrix or the raw DSL. A
					    two-state segmented control — aria-pressed, not tab roles,
					    since the panels below are plain content, not tabpanels. */}
					<div
						role="group"
						aria-label="Model editor mode"
						className="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-1"
					>
						<button
							type="button"
							aria-pressed={mode === 'simple'}
							onClick={() => setMode('simple')}
							className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
								mode === 'simple'
									? 'bg-white text-gray-900 shadow-sm'
									: 'text-gray-500 hover:text-gray-700'
							}`}
						>
							Roles &amp; permissions
						</button>
						<button
							type="button"
							aria-pressed={mode === 'advanced'}
							onClick={() => setMode('advanced')}
							className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
								mode === 'advanced'
									? 'bg-white text-gray-900 shadow-sm'
									: 'text-gray-500 hover:text-gray-700'
							}`}
						>
							Advanced (DSL)
						</button>
					</div>

					{mode === 'simple' ? (
						<>
							<Example>
								<strong>The simplest way to start:</strong> list your roles and
								the actions they can take, then tick who can do what. We turn it
								into a working authorization model for you — no syntax to learn.
								Need hierarchies, groups or conditions? Switch to{' '}
								<strong>Advanced (DSL)</strong>.
							</Example>
							<RbacBuilder
								initialRoles={[]}
								onApply={(generatedDsl) => {
									// Populate the editor and switch to Advanced for review.
									// Saving (a new immutable model version) stays an explicit
									// click on "Save model", so the user controls when a
									// version is created — no churn from repeated clicks.
									setDsl(generatedDsl);
									setValidationError('');
									setMode('advanced');
									toast.success('Model ready — review it below, then Save');
								}}
							/>
						</>
					) : (
						<>
							<Example>
								<strong>Example:</strong> a{' '}
								<code className="rounded bg-white px-1 py-0.5 text-xs">
									document
								</code>{' '}
								has{' '}
								<code className="rounded bg-white px-1 py-0.5 text-xs">
									viewer
								</code>{' '}
								and{' '}
								<code className="rounded bg-white px-1 py-0.5 text-xs">
									editor
								</code>{' '}
								relations, and{' '}
								<code className="rounded bg-white px-1 py-0.5 text-xs">
									can_view = viewer or editor
								</code>
								. Start from an example below and edit it.
							</Example>

							{/* Example catalog opens in a modal — see the dialog below. */}
							<div className="flex flex-wrap items-center gap-2">
								<Button
									type="button"
									variant="outline"
									size="sm"
									onClick={() => setExampleOpen(true)}
								>
									<LayoutTemplate className="mr-2 h-4 w-4" aria-hidden="true" />
									Browse examples
								</Button>
								<span className="text-xs text-gray-500">
									Prebuilt models you can drop in and edit.
								</span>
							</div>

							{/* The model */}
							<div>
								<div className="mb-1.5 flex items-center justify-between">
									<label
										htmlFor="model-dsl"
										className="text-sm font-medium text-gray-700"
									>
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
									<Info
										className="mt-0.5 h-4 w-4 shrink-0 text-gray-400"
										aria-hidden="true"
									/>
									<div>
										<strong className="text-gray-700">
											About model versions.
										</strong>{' '}
										There is always exactly one <em>active</em> model. Saving
										creates a new <strong>immutable version</strong> and makes
										it active; earlier versions are retained so requests already
										in flight stay valid. OpenFGA models are{' '}
										<strong>append-only</strong> — an individual version cannot
										be deleted. To change the rules, save a new version; to
										remove everything, reset the store (deletes the model and
										all tuples). Separate models require separate stores, which
										aren&rsquo;t exposed here.
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
									<span className="whitespace-pre-wrap break-words">
										{validationError}
									</span>
								</div>
							)}
						</>
					)}

					<DocsLinks />

					{modelId && (
						<div className="rounded-lg border border-red-200 bg-red-50/40 p-4">
							<div className="flex items-start gap-2">
								<AlertTriangle
									className="mt-0.5 h-4 w-4 shrink-0 text-red-500"
									aria-hidden="true"
								/>
								<div className="flex-1">
									<p className="text-sm font-medium text-red-800">
										Danger zone · Reset model
									</p>
									<p className="mt-1 text-xs leading-relaxed text-red-700/80">
										OpenFGA models are append-only — individual versions
										can&rsquo;t be deleted. Resetting is the only way to remove
										the model and <strong>all its past versions</strong> and
										start over. This cannot be undone.
									</p>
									{tuplesExist ? (
										<p className="mt-2 flex flex-wrap items-center gap-1 text-xs text-red-700/80">
											<span>
												Reset is blocked while relationship tuples exist, so
												live grants are never dropped silently.
											</span>
											<Link
												to="/authorization/tuples"
												className="font-medium text-red-700 underline underline-offset-2 hover:text-red-800"
											>
												Remove all tuples first
											</Link>
										</p>
									) : (
										<div className="mt-3">
											<Button
												variant="destructive"
												size="sm"
												onClick={() => {
													setConfirmText('');
													setResetOpen(true);
												}}
											>
												<RotateCcw className="mr-2 h-4 w-4" />
												Reset model
											</Button>
										</div>
									)}
								</div>
							</div>
						</div>
					)}

					<div className="flex items-center justify-between border-t border-gray-100 pt-4 text-sm text-gray-500">
						<span>
							Save the model, then grant access with relationship tuples.
						</span>
						<NextStep to="/authorization/tuples" label="Next: grant access" />
					</div>
				</div>
			)}

			<Dialog open={exampleOpen} onOpenChange={setExampleOpen}>
				<DialogContent className="max-w-2xl">
					<DialogHeader>
						<DialogTitle>Start from an example</DialogTitle>
						<DialogDescription>
							Pick a prebuilt model to load into the editor. You can edit it
							before saving — nothing is saved until you click{' '}
							<strong>Save model</strong>.
						</DialogDescription>
					</DialogHeader>
					<div className="grid max-h-[60vh] grid-cols-1 gap-2 overflow-y-auto sm:grid-cols-2">
						{examples.map((ex) => (
							<button
								key={ex.name}
								type="button"
								onClick={() => {
									applyExample(ex.name, ex.dsl);
									setExampleOpen(false);
								}}
								className="rounded-xl border border-gray-200 bg-white p-3 text-left transition-colors hover:border-blue-300 hover:bg-blue-50"
							>
								<span className="block text-sm font-medium text-gray-800">
									{ex.name}
								</span>
								<span className="mt-0.5 block text-xs leading-relaxed text-gray-500">
									{ex.description}
								</span>
							</button>
						))}
					</div>
				</DialogContent>
			</Dialog>

			<Dialog
				open={resetOpen}
				onOpenChange={(open) => !resetting && setResetOpen(open)}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2 text-red-700">
							<AlertTriangle className="h-5 w-5" aria-hidden="true" />
							Reset authorization model
						</DialogTitle>
						<DialogDescription>
							This permanently deletes the active model and every past version,
							then starts a fresh, empty store. You&rsquo;ll need to define a
							new model before access checks work again. This action cannot be
							undone.
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-2">
						<label
							htmlFor="reset-confirm"
							className="text-sm font-medium text-gray-700"
						>
							Type{' '}
							<span className="font-mono font-semibold text-red-700">
								RESET
							</span>{' '}
							to confirm
						</label>
						<Input
							id="reset-confirm"
							value={confirmText}
							onChange={(e) => setConfirmText(e.target.value)}
							placeholder="RESET"
							autoComplete="off"
							spellCheck={false}
						/>
					</div>
					<DialogFooter>
						<DialogClose asChild>
							<Button variant="outline" disabled={resetting}>
								Cancel
							</Button>
						</DialogClose>
						<Button
							variant="destructive"
							onClick={handleReset}
							disabled={resetting || confirmText.trim() !== 'RESET'}
						>
							{resetting ? 'Resetting…' : 'Reset model'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
};

export default Model;
