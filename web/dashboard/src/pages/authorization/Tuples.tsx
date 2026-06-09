import React, { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useClient } from 'urql';
import { toast } from 'sonner';
import {
	Link2,
	Plus,
	Trash2,
	ChevronRight,
	RotateCcw,
	AlertTriangle,
	ArrowRight,
	LayoutTemplate,
} from 'lucide-react';
import { FgaReadTuplesQuery, FgaGetModelQuery } from '../../graphql/queries';
import { FgaWriteTuples, FgaDeleteTuples } from '../../graphql/mutation';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Label } from '../../components/ui/label';
import { Skeleton } from '../../components/ui/skeleton';
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
} from '../../components/ui/dialog';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from '../../components/ui/table';
import FgaNotEnabled from '../../components/FgaNotEnabled';
import AuthSteps, { Example, NextStep } from './AuthSteps';
import DocsLinks from './DocsLinks';
import { isFgaNotEnabledError } from '../../lib/utils';
import type {
	FgaTuple,
	FgaReadTuplesResponse,
	FgaGetModelResponse,
	FgaWriteTuplesResponse,
	FgaDeleteTuplesResponse,
} from '../../types';

// OpenFGA rejects tuple writes until a model exists ("No authorization models
// found for store"). We translate that into a clear, actionable message.
const isNoModelError = (message: string): boolean =>
	/no authorization model/i.test(message);

const PAGE_SIZE = 25;

const emptyForm = { user: '', relation: '', object: '' };

// Common grant patterns. Clicking one fills the form so you can see the shape.
// (Each requires the model to support it — e.g. role usersets, user:* wildcard,
// or a parent relation.)
const GRANT_PATTERNS: {
	name: string;
	desc: string;
	tuple: typeof emptyForm;
}[] = [
	{
		name: 'Direct grant',
		desc: 'One user → one object.',
		tuple: { user: 'user:alice', relation: 'viewer', object: 'document:1' },
	},
	{
		name: 'Assign a role',
		desc: 'Put a user into a role.',
		tuple: { user: 'user:alice', relation: 'assignee', object: 'role:editor' },
	},
	{
		name: 'Grant a whole role',
		desc: 'Everyone in role:editor becomes editor of the object.',
		tuple: {
			user: 'role:editor#assignee',
			relation: 'editor',
			object: 'document:1',
		},
	},
	{
		name: 'Public — all users',
		desc: 'Anyone can access this object (needs user:* in the model).',
		tuple: { user: 'user:*', relation: 'viewer', object: 'document:1' },
	},
	{
		name: 'All resources in a folder',
		desc: 'Grant on the folder once; every document under it inherits.',
		tuple: { user: 'user:alice', relation: 'viewer', object: 'folder:root' },
	},
];

const Tuples = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(true);
	const [fgaDisabled, setFgaDisabled] = useState<boolean>(false);
	const [tuples, setTuples] = useState<FgaTuple[]>([]);
	// continuation token of the page currently displayed.
	const [currentToken, setCurrentToken] = useState<string>('');
	// continuation token to load the next page (empty when exhausted).
	const [nextToken, setNextToken] = useState<string>('');
	// stack of tokens for previous pages so we can page backwards.
	const [tokenStack, setTokenStack] = useState<string[]>([]);
	const [form, setForm] = useState<typeof emptyForm>(emptyForm);
	const [submitting, setSubmitting] = useState<boolean>(false);
	// Tuples can only be written once a model exists. null = not checked yet.
	const [modelExists, setModelExists] = useState<boolean | null>(null);
	// The grant-pattern catalog lives in a modal so the form stays the focus.
	const [patternsOpen, setPatternsOpen] = useState(false);

	const checkModel = useCallback(async () => {
		try {
			const res = await client
				.query<FgaGetModelResponse>(
					FgaGetModelQuery,
					{},
					{ requestPolicy: 'network-only' },
				)
				.toPromise();
			if (res.error) {
				if (isFgaNotEnabledError(res.error)) {
					setFgaDisabled(true);
				} else if (isNoModelError(res.error.message)) {
					// Older servers error instead of returning an empty model.
					setModelExists(false);
				}
				// Any other error (network, transient): leave modelExists unknown so
				// we never wrongly block writes behind a misleading banner.
				return;
			}
			setModelExists(!!res.data?._fga_get_model?.dsl);
		} catch {
			// Leave unknown; the write path still guards with a friendly error.
		}
	}, [client]);

	const fetchTuples = useCallback(
		async (continuationToken: string) => {
			setLoading(true);
			try {
				const res = await client
					.query<FgaReadTuplesResponse>(
						FgaReadTuplesQuery,
						{
							params: {
								page_size: PAGE_SIZE,
								continuation_token: continuationToken || undefined,
							},
						},
						{ requestPolicy: 'network-only' },
					)
					.toPromise();

				if (res.error) {
					if (isFgaNotEnabledError(res.error)) {
						setFgaDisabled(true);
					} else {
						toast.error('Failed to load relationship tuples');
					}
					return;
				}

				if (res.data?._fga_read_tuples) {
					setTuples(res.data._fga_read_tuples.tuples || []);
					setNextToken(res.data._fga_read_tuples.continuation_token || '');
				}
			} catch {
				toast.error('Failed to load relationship tuples');
			} finally {
				setLoading(false);
			}
		},
		[client],
	);

	useEffect(() => {
		fetchTuples('');
		checkModel();
	}, [fetchTuples, checkModel]);

	const goNext = () => {
		if (!nextToken) {
			return;
		}
		setTokenStack((prev) => [...prev, currentToken]);
		setCurrentToken(nextToken);
		fetchTuples(nextToken);
	};

	const goReset = () => {
		setTokenStack([]);
		setCurrentToken('');
		fetchTuples('');
	};

	const handleAdd = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!form.user.trim() || !form.relation.trim() || !form.object.trim()) {
			toast.error('user, relation and object are all required');
			return;
		}
		setSubmitting(true);
		try {
			const res = await client
				.mutation<FgaWriteTuplesResponse>(FgaWriteTuples, {
					params: {
						tuples: [
							{
								user: form.user.trim(),
								relation: form.relation.trim(),
								object: form.object.trim(),
							},
						],
					},
				})
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) {
					setFgaDisabled(true);
				} else if (isNoModelError(res.error.message)) {
					setModelExists(false);
					toast.error('Define and save an authorization model in Step 1 first');
				} else {
					toast.error(res.error.message.replace('[GraphQL] ', ''));
				}
				return;
			}

			toast.success('Tuple added');
			setForm(emptyForm);
			setModelExists(true);
			goReset();
		} catch {
			toast.error('Failed to add tuple');
		} finally {
			setSubmitting(false);
		}
	};

	const handleDelete = async (tuple: FgaTuple) => {
		try {
			const res = await client
				.mutation<FgaDeleteTuplesResponse>(FgaDeleteTuples, {
					params: {
						tuples: [
							{
								user: tuple.user,
								relation: tuple.relation,
								object: tuple.object,
							},
						],
					},
				})
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) {
					setFgaDisabled(true);
				} else {
					toast.error(res.error.message.replace('[GraphQL] ', ''));
				}
				return;
			}

			toast.success('Tuple deleted');
			fetchTuples(currentToken);
		} catch {
			toast.error('Failed to delete tuple');
		}
	};

	if (fgaDisabled) {
		return (
			<div className="m-5 rounded-md bg-white py-5 px-10">
				<AuthSteps current={2} />
				<div className="my-4">
					<h1 className="text-2xl font-semibold text-gray-900">
						Step 2 · Grant access
					</h1>
				</div>
				<FgaNotEnabled />
			</div>
		);
	}

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<AuthSteps current={2} />
			<div className="my-4">
				<h1 className="text-2xl font-semibold text-gray-900">
					Step 2 · Grant access
				</h1>
				<p className="mt-1 max-w-2xl text-sm text-gray-500">
					Grant access by adding a <strong>relationship tuple</strong> — it
					links a <strong>user</strong> to an <strong>object</strong> via a{' '}
					<strong>relation</strong> from your model. Add or remove tuples any
					time to change who has access.
				</p>
			</div>

			{modelExists === false && (
				<div className="mb-4 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-4">
					<AlertTriangle
						className="mt-0.5 h-5 w-5 shrink-0 text-amber-500"
						aria-hidden="true"
					/>
					<div className="flex-1">
						<p className="text-sm font-medium text-amber-800">
							Define a model before granting access
						</p>
						<p className="mt-1 text-xs leading-relaxed text-amber-700/90">
							Relationship tuples reference the relations in your authorization
							model, so OpenFGA requires a saved model before any tuple can be
							written. Head to Step 1, save a model, then come back here.
						</p>
						<Link
							to="/authorization/model"
							className="mt-2 inline-flex items-center gap-1.5 text-sm font-medium text-amber-800 underline underline-offset-2 hover:text-amber-900"
						>
							Go to Step 1 · Define the model
							<ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
						</Link>
					</div>
				</div>
			)}

			<div className="mb-4">
				<Example>
					<strong>Example:</strong> give{' '}
					<code className="rounded bg-white px-1 py-0.5 text-xs">
						user:alice
					</code>{' '}
					the{' '}
					<code className="rounded bg-white px-1 py-0.5 text-xs">viewer</code>{' '}
					relation on{' '}
					<code className="rounded bg-white px-1 py-0.5 text-xs">
						document:1
					</code>{' '}
					— now Alice can view that document.
				</Example>
			</div>

			{/* Common grant patterns — click to prefill the form */}
			{/* Grant-pattern catalog opens in a modal — see the dialog below. */}
			<div className="mb-4 flex flex-wrap items-center gap-2">
				<Button
					type="button"
					variant="outline"
					size="sm"
					onClick={() => setPatternsOpen(true)}
				>
					<LayoutTemplate className="mr-2 h-4 w-4" aria-hidden="true" />
					Browse grant patterns
				</Button>
				<span className="text-xs text-gray-500">
					Common ways to grant access — click one to fill the form.
				</span>
			</div>

			<Dialog open={patternsOpen} onOpenChange={setPatternsOpen}>
				<DialogContent className="max-w-2xl">
					<DialogHeader>
						<DialogTitle>Common grant patterns</DialogTitle>
						<DialogDescription>
							Pick a pattern to fill the form below. Nothing is granted until
							you click <strong>Add</strong>.
						</DialogDescription>
					</DialogHeader>
					<div className="grid max-h-[60vh] grid-cols-1 gap-2 overflow-y-auto sm:grid-cols-2">
						{GRANT_PATTERNS.map((p) => (
							<button
								key={p.name}
								type="button"
								onClick={() => {
									setForm(p.tuple);
									setPatternsOpen(false);
								}}
								className="rounded-xl border border-gray-200 bg-white p-3 text-left transition-colors hover:border-blue-300 hover:bg-blue-50"
							>
								<span className="block text-sm font-medium text-gray-800">
									{p.name}
								</span>
								<span className="mt-0.5 block text-xs leading-relaxed text-gray-500">
									{p.desc}
								</span>
								<span className="mt-1.5 block truncate font-mono text-[11px] text-blue-600">
									{p.tuple.user} · {p.tuple.relation} · {p.tuple.object}
								</span>
							</button>
						))}
					</div>
					<p className="text-xs text-gray-400">
						<strong className="text-gray-500">Tip:</strong> to avoid a tuple per
						object id, grant on a{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5">folder</code>/
						<code className="rounded bg-gray-100 px-1 py-0.5">
							organization
						</code>{' '}
						and let resources inherit, or use{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5">user:*</code> for
						public access.
					</p>
				</DialogContent>
			</Dialog>

			<div className="mb-4">
				<DocsLinks />
			</div>

			{/* Add tuple form */}
			<form
				onSubmit={handleAdd}
				className="mb-6 grid grid-cols-1 gap-3 rounded-md border border-gray-100 bg-gray-50 p-4 md:grid-cols-[1fr_1fr_1fr_auto] md:items-end"
			>
				<div className="space-y-1">
					<Label htmlFor="tuple-user">User</Label>
					<Input
						id="tuple-user"
						placeholder="user:alice"
						value={form.user}
						onChange={(e) => setForm({ ...form, user: e.target.value })}
					/>
				</div>
				<div className="space-y-1">
					<Label htmlFor="tuple-relation">Relation</Label>
					<Input
						id="tuple-relation"
						placeholder="viewer"
						value={form.relation}
						onChange={(e) => setForm({ ...form, relation: e.target.value })}
					/>
				</div>
				<div className="space-y-1">
					<Label htmlFor="tuple-object">Object</Label>
					<Input
						id="tuple-object"
						placeholder="document:1"
						value={form.object}
						onChange={(e) => setForm({ ...form, object: e.target.value })}
					/>
				</div>
				<Button type="submit" disabled={submitting || modelExists === false}>
					<Plus className="mr-2 h-4 w-4" />
					{submitting ? 'Adding...' : 'Add'}
				</Button>
			</form>

			{loading ? (
				<div className="space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : tuples.length ? (
				<>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>User</TableHead>
								<TableHead>Relation</TableHead>
								<TableHead>Object</TableHead>
								<TableHead className="text-right">Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{tuples.map((tuple) => (
								<TableRow
									key={`${tuple.user}|${tuple.relation}|${tuple.object}`}
								>
									<TableCell className="font-mono text-sm">
										{tuple.user}
									</TableCell>
									<TableCell className="font-mono text-sm">
										{tuple.relation}
									</TableCell>
									<TableCell className="font-mono text-sm">
										{tuple.object}
									</TableCell>
									<TableCell className="text-right">
										<Button
											variant="ghost"
											size="sm"
											onClick={() => handleDelete(tuple)}
										>
											<Trash2 className="h-4 w-4 text-red-500" />
										</Button>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>

					{/* Continuation-token pagination */}
					<div className="mt-4 flex items-center justify-between">
						<Button
							variant="outline"
							size="sm"
							onClick={goReset}
							disabled={tokenStack.length === 0 && !currentToken}
						>
							<RotateCcw className="mr-2 h-4 w-4" />
							First page
						</Button>
						<Button
							variant="outline"
							size="sm"
							onClick={goNext}
							disabled={!nextToken}
						>
							Next page
							<ChevronRight className="ml-2 h-4 w-4" />
						</Button>
					</div>
				</>
			) : (
				<div className="flex min-h-[30vh] flex-col items-center justify-center px-4 text-center">
					<div className="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-blue-50">
						<Link2 className="h-6 w-6 text-blue-600" aria-hidden="true" />
					</div>
					<p className="text-base font-semibold text-gray-900">
						No relationship tuples yet
					</p>
					<p className="mt-1 max-w-sm text-sm leading-relaxed text-gray-500">
						Tuples grant access &mdash; e.g.{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5 text-xs text-gray-700">
							user:alice
						</code>{' '}
						is{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5 text-xs text-gray-700">
							viewer
						</code>{' '}
						of{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5 text-xs text-gray-700">
							document:1
						</code>
						. Add one above to grant your first permission.
					</p>
				</div>
			)}

			<div className="mt-6 flex items-center justify-between border-t border-gray-100 pt-4 text-sm text-gray-500">
				<span>Granted some access? Verify it works.</span>
				<NextStep to="/authorization/tester" label="Next: test access" />
			</div>
		</div>
	);
};

export default Tuples;
