import React, { createContext, useContext, useEffect, useRef, useState } from 'react';
import { Tree } from 'react-arborist';
import type { NodeRendererProps } from 'react-arborist';
import { ChevronRight, ChevronDown, Trash2, Plus, X, Boxes, Tag } from 'lucide-react';
import { Input } from '../../components/ui/input';
import { Button } from '../../components/ui/button';
import { Label } from '../../components/ui/label';
import {
	type ModelDraft,
	type ComputedTerm,
	addType,
	deleteType,
	renameType,
	addRelation,
	deleteRelation,
	renameRelation,
	addAssignable,
	removeAssignable,
	addComputed,
	removeComputed,
	relationExprText,
} from './modelDsl';

type Sel = { ti: number; ri: number | null } | null;

interface NodeData {
	id: string;
	name: string;
	kind: 'type' | 'relation';
	ti: number;
	ri?: number;
	expr?: string;
	children?: NodeData[];
}

interface TreeActions {
	onSelect: (sel: Sel) => void;
	onDelete: (d: NodeData) => void;
	onAddRelation: (ti: number) => void;
	selId: string | null;
}
const Ctx = createContext<TreeActions>({
	onSelect: () => {},
	onDelete: () => {},
	onAddRelation: () => {},
	selId: null,
});

const selToId = (sel: Sel) => (sel ? (sel.ri === null ? `t${sel.ti}` : `t${sel.ti}.r${sel.ri}`) : null);

function buildData(model: ModelDraft): NodeData[] {
	return model.types.map((t, ti) => ({
		id: `t${ti}`,
		kind: 'type',
		name: t.name || '(unnamed type)',
		ti,
		children: t.relations.map((r, ri) => ({
			id: `t${ti}.r${ri}`,
			kind: 'relation',
			name: r.name || '(unnamed)',
			ti,
			ri,
			expr: relationExprText(r),
		})),
	}));
}

// Tree row renderer.
function Node({ node, style, dragHandle }: NodeRendererProps<NodeData>) {
	const { onSelect, onDelete, onAddRelation, selId } = useContext(Ctx);
	const d = node.data;
	const isType = d.kind === 'type';
	const selected = node.id === selId;
	return (
		<div ref={dragHandle} style={style}>
			<div
				className={`group flex h-9 items-center gap-1 rounded-md pr-1 ${
					selected ? 'bg-blue-50 ring-1 ring-blue-200' : 'hover:bg-gray-50'
				}`}
				style={{ paddingLeft: node.level * 18 + 4 }}
			>
				{isType ? (
					<button
						type="button"
						onClick={() => node.toggle()}
						className="rounded p-0.5 text-gray-400 hover:text-gray-600"
						aria-label={node.isOpen ? 'Collapse' : 'Expand'}
					>
						{node.isOpen ? (
							<ChevronDown className="h-4 w-4" />
						) : (
							<ChevronRight className="h-4 w-4" />
						)}
					</button>
				) : (
					<span className="w-5" />
				)}
				{isType ? (
					<Boxes className="h-4 w-4 shrink-0 text-gray-400" />
				) : (
					<Tag className="h-3.5 w-3.5 shrink-0 text-gray-400" />
				)}
				<button
					type="button"
					onClick={() => onSelect({ ti: d.ti, ri: d.ri ?? null })}
					className="flex min-w-0 flex-1 items-center gap-2 py-1 text-left"
				>
					<span
						className={`truncate text-sm ${
							isType ? 'font-medium text-gray-800' : 'text-gray-700'
						}`}
					>
						{d.name}
					</span>
					{!isType && (
						<span className="truncate font-mono text-xs text-gray-400">= {d.expr}</span>
					)}
				</button>
				<div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
					{isType && (
						<button
							type="button"
							onClick={() => onAddRelation(d.ti)}
							title="Add relation"
							aria-label="Add relation"
							className="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-700"
						>
							<Plus className="h-3.5 w-3.5" />
						</button>
					)}
					<button
						type="button"
						onClick={() => onDelete(d)}
						title="Delete"
						aria-label={`Delete ${d.name}`}
						className="rounded p-1 text-gray-400 hover:bg-red-100 hover:text-red-600"
					>
						<Trash2 className="h-3.5 w-3.5" />
					</button>
				</div>
			</div>
		</div>
	);
}

const Chip = ({ label, onRemove }: { label: string; onRemove: () => void }) => (
	<span className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
		<span className="font-mono">{label}</span>
		<button type="button" onClick={onRemove} aria-label={`Remove ${label}`} className="text-blue-400 hover:text-blue-700">
			<X className="h-3 w-3" />
		</button>
	</span>
);

const selectCls =
	'rounded-md border border-gray-200 bg-white px-2 py-1 text-xs text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400';

// Detail pane: edits the selected type or relation.
const DetailPane = ({
	model,
	sel,
	onChange,
}: {
	model: ModelDraft;
	sel: Sel;
	onChange: (m: ModelDraft) => void;
}) => {
	const [compRel, setCompRel] = useState('');
	const [compFrom, setCompFrom] = useState('');

	if (!sel || !model.types[sel.ti]) {
		return (
			<div className="flex min-h-[200px] items-center justify-center rounded-xl border border-dashed border-gray-200 p-6 text-center text-sm text-gray-400">
				Select a type or relation on the left to edit it.
			</div>
		);
	}

	const t = model.types[sel.ti];

	if (sel.ri === null) {
		return (
			<div className="rounded-xl border border-gray-200 bg-white p-4">
				<div className="mb-3 text-xs font-medium uppercase tracking-wide text-gray-400">Type</div>
				<Label htmlFor="type-name">Name</Label>
				<Input
					id="type-name"
					value={t.name}
					onChange={(e) => onChange(renameType(model, sel.ti, e.target.value.trim()))}
					placeholder="document"
					className="mt-1 font-mono text-sm"
				/>
				<p className="mt-3 text-xs leading-relaxed text-gray-500">
					{t.relations.length} relation{t.relations.length === 1 ? '' : 's'}. Use the{' '}
					<Plus className="inline h-3 w-3" /> on this type in the tree to add one.
				</p>
			</div>
		);
	}

	const r = t.relations[sel.ri];
	if (!r) return null;
	const typeNames = model.types.map((x) => x.name).filter(Boolean);
	const relNames = t.relations.map((x) => x.name).filter((n) => n && n !== r.name);

	return (
		<div className="space-y-4 rounded-xl border border-gray-200 bg-white p-4">
			<div>
				<div className="mb-3 text-xs font-medium uppercase tracking-wide text-gray-400">
					Relation on <span className="font-mono lowercase text-gray-500">{t.name}</span>
				</div>
				<Label htmlFor="rel-name">Name</Label>
				<Input
					id="rel-name"
					value={r.name}
					onChange={(e) => onChange(renameRelation(model, sel.ti, sel.ri!, e.target.value.trim()))}
					placeholder="viewer"
					className="mt-1 font-mono text-sm"
				/>
			</div>

			<div>
				<p className="mb-1.5 text-xs font-medium text-gray-600">Directly assignable to</p>
				<div className="flex flex-wrap items-center gap-1.5">
					{r.directTypes.map((dt, idx) => (
						<Chip key={idx} label={dt} onRemove={() => onChange(removeAssignable(model, sel.ti, sel.ri!, idx))} />
					))}
					<select
						value=""
						onChange={(e) => e.target.value && onChange(addAssignable(model, sel.ti, sel.ri!, e.target.value))}
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

			<div>
				<p className="mb-1.5 text-xs font-medium text-gray-600">Or computed from (union)</p>
				<div className="flex flex-wrap items-center gap-1.5">
					{r.computed.map((c, idx) => (
						<Chip
							key={idx}
							label={c.from ? `${c.relation} from ${c.from}` : c.relation}
							onRemove={() => onChange(removeComputed(model, sel.ti, sel.ri!, idx))}
						/>
					))}
					<span className="inline-flex items-center gap-1">
						<select value={compRel} onChange={(e) => setCompRel(e.target.value)} className={selectCls}>
							<option value="">+ relation…</option>
							{relNames.map((n) => (
								<option key={n} value={n}>
									{n}
								</option>
							))}
						</select>
						{compRel && (
							<>
								<select value={compFrom} onChange={(e) => setCompFrom(e.target.value)} className={selectCls}>
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
										const term: ComputedTerm = compFrom
											? { relation: compRel, from: compFrom }
											: { relation: compRel };
										onChange(addComputed(model, sel.ti, sel.ri!, term));
										setCompRel('');
										setCompFrom('');
									}}
									className="rounded-md bg-blue-500 px-2 py-1 text-xs font-medium text-white hover:bg-blue-600"
								>
									Add
								</button>
							</>
						)}
					</span>
				</div>
			</div>

			<p className="border-t border-gray-100 pt-3 font-mono text-xs text-gray-500">
				define {r.name || '…'}: {relationExprText(r)}
			</p>
		</div>
	);
};

interface Props {
	model: ModelDraft;
	onChange: (m: ModelDraft) => void;
}

const ModelTree = ({ model, onChange }: Props) => {
	const [sel, setSel] = useState<Sel>(null);
	const containerRef = useRef<HTMLDivElement>(null);
	const [width, setWidth] = useState(360);

	useEffect(() => {
		const el = containerRef.current;
		if (!el) return;
		const ro = new ResizeObserver((entries) => {
			for (const e of entries) setWidth(e.contentRect.width);
		});
		ro.observe(el);
		return () => ro.disconnect();
	}, []);

	const data = buildData(model);
	const rowCount = data.reduce((n, t) => n + 1 + (t.children?.length || 0), 0);
	const height = Math.min(Math.max(rowCount * 36 + 8, 160), 480);

	const actions: TreeActions = {
		selId: selToId(sel),
		onSelect: setSel,
		onDelete: (d) => {
			onChange(d.kind === 'type' ? deleteType(model, d.ti) : deleteRelation(model, d.ti, d.ri!));
			setSel(null);
		},
		onAddRelation: (ti) => {
			setSel({ ti, ri: model.types[ti].relations.length });
			onChange(addRelation(model, ti));
		},
	};

	return (
		<div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
			<div className="rounded-xl border border-gray-200 bg-white p-2">
				<div ref={containerRef} className="min-h-[160px]">
					<Ctx.Provider value={actions}>
						<Tree<NodeData>
							data={data}
							openByDefault
							width={width}
							height={height}
							indent={18}
							rowHeight={36}
							disableDrag
							disableMultiSelection
						>
							{Node}
						</Tree>
					</Ctx.Provider>
				</div>
				<Button
					variant="outline"
					size="sm"
					className="mt-2"
					onClick={() => {
						setSel({ ti: model.types.length, ri: null });
						onChange(addType(model));
					}}
				>
					<Plus className="mr-1 h-3.5 w-3.5" /> Add type
				</Button>
			</div>

			<DetailPane key={actions.selId ?? 'none'} model={model} sel={sel} onChange={onChange} />
		</div>
	);
};

export default ModelTree;
