import React, { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useClient } from 'urql';
import { Check, ArrowRight, Lightbulb } from 'lucide-react';
import { FgaGetModelQuery, FgaReadTuplesQuery } from '../../graphql/queries';
import type { FgaGetModelResponse, FgaReadTuplesResponse } from '../../types';

// The three FGA pages are a sequential workflow: define the model, grant access,
// then verify. AuthSteps renders that as a clickable stepper shown on each page,
// so the flow is always visible but any step is reachable directly.
export const STEPS = [
	{ n: 1, label: 'Define model', desc: 'Set the rules', route: '/authorization/model' },
	{ n: 2, label: 'Grant access', desc: 'Who can do what', route: '/authorization/tuples' },
	{ n: 3, label: 'Test access', desc: 'Verify it works', route: '/authorization/tester' },
] as const;

const AuthSteps = ({ current }: { current: 1 | 2 | 3 }) => {
	const navigate = useNavigate();
	const client = useClient();
	// A step shows a check only when it's actually complete: step 1 when a model
	// is saved, step 2 when at least one tuple exists. Best-effort; ignore errors.
	const [modelDone, setModelDone] = useState(false);
	const [tuplesDone, setTuplesDone] = useState(false);

	useEffect(() => {
		let active = true;
		client
			.query<FgaGetModelResponse>(FgaGetModelQuery, {}, { requestPolicy: 'network-only' })
			.toPromise()
			.then((r) => active && setModelDone(!!r.data?._fga_get_model?.dsl))
			.catch(() => {});
		client
			.query<FgaReadTuplesResponse>(
				FgaReadTuplesQuery,
				{ params: { page_size: 1 } },
				{ requestPolicy: 'network-only' },
			)
			.toPromise()
			.then((r) => active && setTuplesDone((r.data?._fga_read_tuples?.tuples?.length ?? 0) > 0))
			.catch(() => {});
		return () => {
			active = false;
		};
	}, [client]);

	const isDone = (n: number) => (n === 1 ? modelDone : n === 2 ? tuplesDone : false);

	return (
		<nav aria-label="Fine-grained authorization setup" className="mb-5">
			<ol className="flex flex-col gap-2 sm:flex-row sm:items-stretch">
				{STEPS.map((s) => {
					const state = s.n === current ? 'current' : isDone(s.n) ? 'done' : 'upcoming';
					return (
						<li key={s.n} className="flex-1">
							<button
								type="button"
								onClick={() => navigate(s.route)}
								aria-current={state === 'current' ? 'step' : undefined}
								className={`flex w-full items-center gap-3 rounded-xl border px-3 py-2.5 text-left transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 ${
									state === 'current'
										? 'border-blue-200 bg-blue-50'
										: 'border-gray-200 bg-white hover:border-gray-300 hover:bg-gray-50'
								}`}
							>
								<span
									className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold ${
										state === 'done'
											? 'bg-green-100 text-green-700'
											: state === 'current'
												? 'bg-blue-600 text-white'
												: 'bg-gray-100 text-gray-500'
									}`}
								>
									{state === 'done' ? <Check className="h-4 w-4" aria-hidden="true" /> : s.n}
								</span>
								<span className="min-w-0">
									<span
										className={`block text-sm font-medium ${
											state === 'current' ? 'text-blue-700' : 'text-gray-700'
										}`}
									>
										{s.label}
									</span>
									<span className="block truncate text-xs text-gray-400">{s.desc}</span>
								</span>
							</button>
						</li>
					);
				})}
			</ol>
		</nav>
	);
};

// Example renders a concrete "for example…" callout, so each step shows what a
// real entry looks like.
export const Example = ({ children }: { children: React.ReactNode }) => (
	<div className="flex items-start gap-2.5 rounded-lg border border-blue-100 bg-blue-50/60 px-3.5 py-2.5 text-sm text-gray-600">
		<Lightbulb className="mt-0.5 h-4 w-4 shrink-0 text-blue-500" aria-hidden="true" />
		<div className="leading-relaxed">{children}</div>
	</div>
);

// NextStep is the primary "continue the flow" link at the bottom of a step.
export const NextStep = ({ to, label }: { to: string; label: string }) => (
	<Link
		to={to}
		className="inline-flex items-center gap-1.5 rounded-lg bg-blue-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
	>
		{label}
		<ArrowRight className="h-4 w-4" aria-hidden="true" />
	</Link>
);

export default AuthSteps;
