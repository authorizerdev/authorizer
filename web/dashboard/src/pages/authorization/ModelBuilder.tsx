import React, { useState } from 'react';
import { Plus, Trash2, X } from 'lucide-react';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import type { ModelDraft, TypeDef, RelationDef, ComputedTerm } from './modelDsl';

const selectCls =
	'rounded-md border border-gray-200 bg-white px-2 py-1 text-xs text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400';

const Chip = ({ label, onRemove }: { label: string; onRemove: () => void }) => (
	<span className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
		<span className="font-mono">{label}</span>
		<button
			type="button"
			onClick={onRemove}
			aria-label={`Remove ${label}`}
			className="text-blue-400 transition-colors hover:text-blue-700"
		>
			<X className="h-3 w-3" />
		</button>
	</span>
);

// ComputedAdder lets the user add one OR-ed computed term: a relation, optionally
// inherited "from" another relation on this type.
const ComputedAdder = ({
	relNames,
	onAdd,
}: {
	relNames: string[];
	onAdd: (t: ComputedTerm) => void;
}) => {
	const [rel, setRel] = useState('');
	const [from, setFrom] = useState('');
	return (
		<span className="inline-flex items-center gap-1">
			<select value={rel} onChange={(e) => setRel(e.target.value)} className={selectCls}>
				<option value="">+ relation…</option>
				{relNames.map((n) => (
					<option key={n} value={n}>
						{n}
					</option>
				))}
			</select>
			{rel && (
				<>
					<select value={from} onChange={(e) => setFrom(e.target.value)} className={selectCls}>
						<option value="">(direct)</option>
						{relNames.map((n) => (
							<option key={n} value={n}>
								from {n}
							</option>
						))}
					</select>
					<button
						type="button"
						onClick={() => {
							onAdd(from ? { relation: rel, from } : { relation: rel });
							setRel('');
							setFrom('');
						}}
						className="rounded-md bg-blue-500 px-2 py-1 text-xs font-medium text-white hover:bg-blue-600"
					>
						Add
					</button>
				</>
			)}
		</span>
	);
};

interface Props {
	model: ModelDraft;
	onChange: (m: ModelDraft) => void;
}

const ModelBuilder = ({ model, onChange }: Props) => {
	const setTypes = (types: TypeDef[]) => onChange({ ...model, types });
	const updateType = (ti: number, patch: Partial<TypeDef>) =>
		setTypes(model.types.map((t, i) => (i === ti ? { ...t, ...patch } : t)));
	const updateRelation = (ti: number, ri: number, patch: Partial<RelationDef>) =>
		updateType(ti, {
			relations: model.types[ti].relations.map((r, i) => (i === ri ? { ...r, ...patch } : r)),
		});

	const typeNames = model.types.map((t) => t.name).filter(Boolean);

	return (
		<div className="space-y-4">
			{model.types.map((t, ti) => {
				const relNames = t.relations.map((r) => r.name).filter(Boolean);
				return (
					<div key={ti} className="rounded-xl border border-gray-200 bg-white p-4">
						<div className="flex items-center gap-3">
							<span className="text-xs font-medium uppercase tracking-wide text-gray-400">
								type
							</span>
							<Input
								value={t.name}
								onChange={(e) => updateType(ti, { name: e.target.value.trim() })}
								placeholder="document"
								className="h-8 max-w-[220px] font-mono text-sm"
							/>
							<Button
								variant="ghost"
								size="sm"
								className="ml-auto"
								onClick={() => setTypes(model.types.filter((_, i) => i !== ti))}
								aria-label="Delete type"
							>
								<Trash2 className="h-4 w-4 text-red-500" />
							</Button>
						</div>

						{t.relations.length > 0 && (
							<div className="mt-3 space-y-3 border-t border-gray-100 pt-3">
								{t.relations.map((r, ri) => (
									<div key={ri} className="rounded-lg bg-gray-50 p-3">
										<div className="flex items-center gap-2">
											<Input
												value={r.name}
												onChange={(e) => updateRelation(ti, ri, { name: e.target.value.trim() })}
												placeholder="viewer"
												className="h-8 max-w-[200px] font-mono text-sm"
											/>
											<Button
												variant="ghost"
												size="sm"
												className="ml-auto"
												onClick={() =>
													updateType(ti, {
														relations: t.relations.filter((_, i) => i !== ri),
													})
												}
												aria-label="Delete relation"
											>
												<Trash2 className="h-4 w-4 text-red-400" />
											</Button>
										</div>

										<div className="mt-2">
											<p className="mb-1 text-xs text-gray-500">Directly assignable to</p>
											<div className="flex flex-wrap items-center gap-1.5">
												{r.directTypes.map((dt, di) => (
													<Chip
														key={di}
														label={dt}
														onRemove={() =>
															updateRelation(ti, ri, {
																directTypes: r.directTypes.filter((_, i) => i !== di),
															})
														}
													/>
												))}
												<select
													value=""
													onChange={(e) => {
														const v = e.target.value;
														if (v && !r.directTypes.includes(v)) {
															updateRelation(ti, ri, { directTypes: [...r.directTypes, v] });
														}
													}}
													className={selectCls}
												>
													<option value="">+ type…</option>
													{typeNames.map((n) => (
														<option key={n} value={n}>
															{n}
														</option>
													))}
												</select>
											</div>
										</div>

										<div className="mt-2">
											<p className="mb-1 text-xs text-gray-500">Or computed from (union)</p>
											<div className="flex flex-wrap items-center gap-1.5">
												{r.computed.map((c, ci) => (
													<Chip
														key={ci}
														label={c.from ? `${c.relation} from ${c.from}` : c.relation}
														onRemove={() =>
															updateRelation(ti, ri, {
																computed: r.computed.filter((_, i) => i !== ci),
															})
														}
													/>
												))}
												<ComputedAdder
													relNames={relNames.filter((n) => n !== r.name)}
													onAdd={(term) =>
														updateRelation(ti, ri, { computed: [...r.computed, term] })
													}
												/>
											</div>
										</div>
									</div>
								))}
							</div>
						)}

						<Button
							variant="outline"
							size="sm"
							className="mt-3"
							onClick={() =>
								updateType(ti, {
									relations: [...t.relations, { name: '', directTypes: [], computed: [] }],
								})
							}
						>
							<Plus className="mr-1 h-3.5 w-3.5" /> Add relation / permission
						</Button>
					</div>
				);
			})}

			<Button variant="outline" onClick={() => setTypes([...model.types, { name: '', relations: [] }])}>
				<Plus className="mr-2 h-4 w-4" /> Add type
			</Button>
		</div>
	);
};

export default ModelBuilder;
