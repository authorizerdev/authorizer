import React, { useState } from 'react';
import { useClient } from 'urql';
import { toast } from 'sonner';
import { Plus, Trash2, ShieldCheck, ShieldX, Play } from 'lucide-react';
import { FgaCheckQuery } from '../../graphql/queries';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Label } from '../../components/ui/label';
import { Badge } from '../../components/ui/badge';
import FgaNotEnabled from '../../components/FgaNotEnabled';
import AuthSteps, { Example } from './AuthSteps';
import { isFgaNotEnabledError } from '../../lib/utils';
import type { FgaTuple, FgaCheckResponse } from '../../types';

const emptyTuple: FgaTuple = { user: '', relation: '', object: '' };

const Tester = () => {
	const client = useClient();
	const [fgaDisabled, setFgaDisabled] = useState<boolean>(false);
	const [relation, setRelation] = useState<string>('');
	const [object, setObject] = useState<string>('');
	const [contextualTuples, setContextualTuples] = useState<FgaTuple[]>([]);
	const [running, setRunning] = useState<boolean>(false);
	const [result, setResult] = useState<boolean | null>(null);

	const addContextualTuple = () => {
		setContextualTuples((prev) => [...prev, { ...emptyTuple }]);
	};

	const updateContextualTuple = (
		index: number,
		field: keyof FgaTuple,
		value: string,
	) => {
		setContextualTuples((prev) =>
			prev.map((t, i) => (i === index ? { ...t, [field]: value } : t)),
		);
	};

	const removeContextualTuple = (index: number) => {
		setContextualTuples((prev) => prev.filter((_, i) => i !== index));
	};

	const handleCheck = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!relation.trim() || !object.trim()) {
			toast.error('relation and object are required');
			return;
		}
		setRunning(true);
		setResult(null);
		try {
			const validContextual = contextualTuples
				.filter(
					(t) => t.user.trim() && t.relation.trim() && t.object.trim(),
				)
				.map((t) => ({
					user: t.user.trim(),
					relation: t.relation.trim(),
					object: t.object.trim(),
				}));

			const params: {
				relation: string;
				object: string;
				contextual_tuples?: FgaTuple[];
			} = {
				relation: relation.trim(),
				object: object.trim(),
			};
			if (validContextual.length) {
				params.contextual_tuples = validContextual;
			}

			const res = await client
				.query<FgaCheckResponse>(
					FgaCheckQuery,
					{ params },
					{ requestPolicy: 'network-only' },
				)
				.toPromise();

			if (res.error) {
				if (isFgaNotEnabledError(res.error)) {
					setFgaDisabled(true);
				} else {
					toast.error(res.error.message.replace('[GraphQL] ', ''));
				}
				return;
			}

			if (res.data?.fga_check) {
				setResult(res.data.fga_check.allowed);
			}
		} catch {
			toast.error('Failed to run access check');
		} finally {
			setRunning(false);
		}
	};

	if (fgaDisabled) {
		return (
			<div className="m-5 rounded-md bg-white py-5 px-10">
				<AuthSteps current={3} />
				<div className="my-4">
					<h1 className="text-2xl font-semibold text-gray-900">Step 3 · Test access</h1>
				</div>
				<FgaNotEnabled />
			</div>
		);
	}

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<AuthSteps current={3} />
			<div className="my-4">
				<h1 className="text-2xl font-semibold text-gray-900">Step 3 · Test access</h1>
				<p className="mt-1 max-w-2xl text-sm text-gray-500">
					Verify your rules and grants. The check runs for{' '}
					<strong>the currently logged-in admin</strong> &mdash; the principal is pinned to your
					token by the server and cannot be changed here.
				</p>
			</div>

			<div className="mb-5">
				<Example>
					<strong>Example:</strong> ask &ldquo;can I{' '}
					<code className="rounded bg-white px-1 py-0.5 text-xs">can_view</code>{' '}
					<code className="rounded bg-white px-1 py-0.5 text-xs">document:1</code>?&rdquo; &mdash; if
					you granted yourself <code className="rounded bg-white px-1 py-0.5 text-xs">viewer</code>{' '}
					on it in step 2, the result is <strong>Allowed</strong>.
				</Example>
			</div>

			<form onSubmit={handleCheck} className="max-w-2xl space-y-5">
				<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
					<div className="space-y-1">
						<Label htmlFor="check-relation">Relation</Label>
						<Input
							id="check-relation"
							placeholder="viewer"
							value={relation}
							onChange={(e) => setRelation(e.target.value)}
						/>
					</div>
					<div className="space-y-1">
						<Label htmlFor="check-object">Object</Label>
						<Input
							id="check-object"
							placeholder="document:1"
							value={object}
							onChange={(e) => setObject(e.target.value)}
						/>
					</div>
				</div>

				{/* Contextual tuples */}
				<div className="space-y-2">
					<div className="flex items-center justify-between">
						<Label>Contextual tuples (optional)</Label>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onClick={addContextualTuple}
						>
							<Plus className="mr-2 h-4 w-4" />
							Add
						</Button>
					</div>
					{contextualTuples.length === 0 ? (
						<p className="text-xs text-gray-400">
							Contextual tuples are evaluated only for this check and are not
							persisted.
						</p>
					) : (
						contextualTuples.map((tuple, index) => (
							<div
								key={index}
								className="grid grid-cols-1 gap-2 md:grid-cols-[1fr_1fr_1fr_auto] md:items-center"
							>
								<Input
									placeholder="user:alice"
									value={tuple.user}
									onChange={(e) =>
										updateContextualTuple(index, 'user', e.target.value)
									}
								/>
								<Input
									placeholder="viewer"
									value={tuple.relation}
									onChange={(e) =>
										updateContextualTuple(index, 'relation', e.target.value)
									}
								/>
								<Input
									placeholder="document:1"
									value={tuple.object}
									onChange={(e) =>
										updateContextualTuple(index, 'object', e.target.value)
									}
								/>
								<Button
									type="button"
									variant="ghost"
									size="sm"
									onClick={() => removeContextualTuple(index)}
								>
									<Trash2 className="h-4 w-4 text-red-500" />
								</Button>
							</div>
						))
					)}
				</div>

				<Button type="submit" disabled={running}>
					<Play className="mr-2 h-4 w-4" />
					{running ? 'Checking...' : 'Run Check'}
				</Button>
			</form>

			{result !== null && (
				<div className="mt-6 max-w-2xl">
					{result ? (
						<div className="flex items-center gap-3 rounded-md border border-green-200 bg-green-50 p-4">
							<ShieldCheck className="h-6 w-6 text-green-600" />
							<div>
								<Badge variant="success">Allowed</Badge>
								<p className="mt-1 text-sm text-gray-600">
									You have <strong>{relation}</strong> access to{' '}
									<strong>{object}</strong>.
								</p>
							</div>
						</div>
					) : (
						<div className="flex items-center gap-3 rounded-md border border-red-200 bg-red-50 p-4">
							<ShieldX className="h-6 w-6 text-red-600" />
							<div>
								<Badge variant="destructive">Denied</Badge>
								<p className="mt-1 text-sm text-gray-600">
									You do not have <strong>{relation}</strong> access to{' '}
									<strong>{object}</strong>.
								</p>
							</div>
						</div>
					)}
				</div>
			)}
		</div>
	);
};

export default Tester;
