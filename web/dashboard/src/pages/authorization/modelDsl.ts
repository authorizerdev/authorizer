// modelDsl.ts — pure helpers to convert between a visual ModelDraft and OpenFGA
// authorization-model DSL. The builder covers the common subset (types, direct
// assignment, unions via `or`, and inheritance via `X from Y`). Advanced
// constructs (`and`, `but not`, conditions `with`, grouping) are not represented
// in the builder — parseDsl reports supported=false for those so the UI can keep
// the user in raw-DSL mode.

// ComputedTerm is one OR-ed term of a computed relation: either another relation
// on the same type ({relation}) or an inherited one ({relation, from}).
export interface ComputedTerm {
	relation: string;
	from?: string;
}

// RelationDef is a single `define <name>: ...`. The effective rule is the union
// (OR) of directTypes (the `[...]` assignable part) and the computed terms.
export interface RelationDef {
	name: string;
	directTypes: string[]; // e.g. ["user"], ["user", "team#member"], ["folder"]
	computed: ComputedTerm[]; // OR-ed with directTypes
}

export interface TypeDef {
	name: string;
	relations: RelationDef[];
}

export interface ModelDraft {
	types: TypeDef[];
}

const IDENT = /^[a-zA-Z0-9_]+$/;

// relationExpr renders the right-hand side of a `define`.
function relationExpr(r: RelationDef): string {
	const parts: string[] = [];
	if (r.directTypes.length) {
		parts.push(`[${r.directTypes.join(', ')}]`);
	}
	for (const c of r.computed) {
		parts.push(c.from ? `${c.relation} from ${c.from}` : c.relation);
	}
	return parts.join(' or ');
}

// generateDsl renders a ModelDraft to OpenFGA DSL.
export function generateDsl(model: ModelDraft): string {
	const lines: string[] = ['model', '  schema 1.1', ''];
	for (const t of model.types) {
		lines.push(`type ${t.name}`);
		if (t.relations.length) {
			lines.push('  relations');
			for (const r of t.relations) {
				lines.push(`    define ${r.name}: ${relationExpr(r)}`);
			}
		}
	}
	return lines.join('\n') + '\n';
}

// parseDsl best-effort parses DSL into a ModelDraft. supported=false means the
// model uses constructs the visual builder can't represent — the caller should
// stay in DSL mode.
export function parseDsl(dsl: string): { model: ModelDraft | null; supported: boolean } {
	const types: TypeDef[] = [];
	let current: TypeDef | null = null;

	for (const raw of dsl.split('\n')) {
		const trimmed = raw.trim();
		if (
			!trimmed ||
			trimmed === 'model' ||
			trimmed.startsWith('schema') ||
			trimmed === 'relations' ||
			trimmed.startsWith('#')
		) {
			continue;
		}

		if (trimmed.startsWith('type ')) {
			current = { name: trimmed.slice(5).trim(), relations: [] };
			types.push(current);
			continue;
		}

		if (trimmed.startsWith('define ')) {
			if (!current) return { model: null, supported: false };
			const m = trimmed.match(/^define\s+([a-zA-Z0-9_]+)\s*:\s*(.+)$/);
			if (!m) return { model: null, supported: false };
			const expr = m[2].trim();
			// Constructs the builder cannot represent.
			if (/\bbut\s+not\b|\band\b|[()]|\bwith\b/.test(expr)) {
				return { model: null, supported: false };
			}
			const rel: RelationDef = { name: m[1], directTypes: [], computed: [] };
			for (const partRaw of expr.split(/\s+or\s+/)) {
				const part = partRaw.trim();
				if (!part) continue;
				if (part.startsWith('[') && part.endsWith(']')) {
					rel.directTypes.push(
						...part
							.slice(1, -1)
							.split(',')
							.map((s) => s.trim())
							.filter(Boolean),
					);
				} else if (/\sfrom\s/.test(part)) {
					const fm = part.match(/^([a-zA-Z0-9_]+)\s+from\s+([a-zA-Z0-9_]+)$/);
					if (!fm) return { model: null, supported: false };
					rel.computed.push({ relation: fm[1], from: fm[2] });
				} else if (IDENT.test(part)) {
					rel.computed.push({ relation: part });
				} else {
					// e.g. a computed userset like team#member — not builder-representable.
					return { model: null, supported: false };
				}
			}
			current.relations.push(rel);
			continue;
		}

		// Unknown non-empty line — be conservative.
		return { model: null, supported: false };
	}

	if (!types.length) return { model: null, supported: false };
	return { model: { types }, supported: true };
}

// validateModel returns a human-readable error for an unsaveable model, or "".
export function validateModel(model: ModelDraft): string {
	if (!model.types.length) return 'Add at least one type.';
	const names = new Set<string>();
	for (const t of model.types) {
		if (!IDENT.test(t.name)) return `Type name "${t.name}" must be alphanumeric/underscore.`;
		if (names.has(t.name)) return `Duplicate type "${t.name}".`;
		names.add(t.name);
		const relNames = new Set<string>();
		for (const r of t.relations) {
			if (!IDENT.test(r.name)) return `Relation "${r.name}" on ${t.name} is not a valid name.`;
			if (relNames.has(r.name)) return `Duplicate relation "${r.name}" on ${t.name}.`;
			relNames.add(r.name);
			if (!r.directTypes.length && !r.computed.length) {
				return `Relation "${r.name}" on ${t.name} needs at least one assignable type or computed relation.`;
			}
		}
	}
	return '';
}

// summarize returns plain-English lines describing the model.
export function summarize(model: ModelDraft): string[] {
	const out: string[] = [];
	for (const t of model.types) {
		for (const r of t.relations) {
			const bits: string[] = [];
			if (r.directTypes.length) bits.push(`assigned (${r.directTypes.join(', ')})`);
			for (const c of r.computed) {
				bits.push(c.from ? `${c.relation} of its ${c.from}` : `a ${c.relation}`);
			}
			out.push(`A user who is ${bits.join(' OR ')} is "${r.name}" of a ${t.name}.`);
		}
	}
	return out;
}

// ── Pure model mutations (kept pure + index-based so they're unit-testable;
//    the tree editor calls these and never mutates state in place). ───────────

const mapType = (m: ModelDraft, ti: number, fn: (t: TypeDef) => TypeDef): ModelDraft => ({
	...m,
	types: m.types.map((t, i) => (i === ti ? fn(t) : t)),
});

const mapRelation = (
	m: ModelDraft,
	ti: number,
	ri: number,
	fn: (r: RelationDef) => RelationDef,
): ModelDraft => mapType(m, ti, (t) => ({ ...t, relations: t.relations.map((r, i) => (i === ri ? fn(r) : r)) }));

export const addType = (m: ModelDraft): ModelDraft => ({
	...m,
	types: [...m.types, { name: '', relations: [] }],
});

export const deleteType = (m: ModelDraft, ti: number): ModelDraft => ({
	...m,
	types: m.types.filter((_, i) => i !== ti),
});

export const renameType = (m: ModelDraft, ti: number, name: string): ModelDraft =>
	mapType(m, ti, (t) => ({ ...t, name }));

export const addRelation = (m: ModelDraft, ti: number): ModelDraft =>
	mapType(m, ti, (t) => ({ ...t, relations: [...t.relations, { name: '', directTypes: [], computed: [] }] }));

export const deleteRelation = (m: ModelDraft, ti: number, ri: number): ModelDraft =>
	mapType(m, ti, (t) => ({ ...t, relations: t.relations.filter((_, i) => i !== ri) }));

export const renameRelation = (m: ModelDraft, ti: number, ri: number, name: string): ModelDraft =>
	mapRelation(m, ti, ri, (r) => ({ ...r, name }));

export const addAssignable = (m: ModelDraft, ti: number, ri: number, dt: string): ModelDraft =>
	mapRelation(m, ti, ri, (r) =>
		r.directTypes.includes(dt) ? r : { ...r, directTypes: [...r.directTypes, dt] },
	);

export const removeAssignable = (m: ModelDraft, ti: number, ri: number, idx: number): ModelDraft =>
	mapRelation(m, ti, ri, (r) => ({ ...r, directTypes: r.directTypes.filter((_, i) => i !== idx) }));

export const addComputed = (m: ModelDraft, ti: number, ri: number, term: ComputedTerm): ModelDraft =>
	mapRelation(m, ti, ri, (r) => ({ ...r, computed: [...r.computed, term] }));

export const removeComputed = (m: ModelDraft, ti: number, ri: number, idx: number): ModelDraft =>
	mapRelation(m, ti, ri, (r) => ({ ...r, computed: r.computed.filter((_, i) => i !== idx) }));

// relationExprText renders a relation's definition for compact display.
export function relationExprText(r: RelationDef): string {
	const parts: string[] = [];
	if (r.directTypes.length) parts.push(`[${r.directTypes.join(', ')}]`);
	for (const c of r.computed) parts.push(c.from ? `${c.relation} from ${c.from}` : c.relation);
	return parts.join(' or ') || '(empty)';
}

// sanitizeRelationName makes an Authorizer role usable as an OpenFGA relation
// name (alphanumeric/underscore). e.g. "org-admin" -> "org_admin".
export function sanitizeRelationName(role: string): string {
	return role.trim().replace(/[^a-zA-Z0-9_]/g, '_').replace(/^_+|_+$/g, '');
}

// rolesTemplate builds a starter model from the instance's configured roles:
// each role becomes a directly-assignable relation on a `resource`, plus a
// `can_access` permission that any of those roles satisfies. A concrete,
// builder-friendly starting point the admin can then refine.
export function rolesTemplate(roles: string[]): ModelDraft | null {
	const seen = new Set<string>();
	const roleRelations: RelationDef[] = [];
	for (const raw of roles) {
		const name = sanitizeRelationName(raw);
		if (!name || seen.has(name)) continue;
		seen.add(name);
		roleRelations.push({ name, directTypes: ['user'], computed: [] });
	}
	if (!roleRelations.length) return null;

	roleRelations.push({
		name: 'can_access',
		directTypes: [],
		computed: roleRelations.map((r) => ({ relation: r.name })),
	});

	return {
		types: [
			{ name: 'user', relations: [] },
			{ name: 'resource', relations: roleRelations },
		],
	};
}

// TEMPLATES are builder-representable starter models.
export const TEMPLATES: { name: string; description: string; model: ModelDraft }[] = [
	{
		name: 'Document sharing',
		description: 'Owner / editor / viewer with cascading permissions.',
		model: {
			types: [
				{ name: 'user', relations: [] },
				{
					name: 'document',
					relations: [
						{ name: 'owner', directTypes: ['user'], computed: [] },
						{ name: 'editor', directTypes: ['user'], computed: [{ relation: 'owner' }] },
						{ name: 'viewer', directTypes: ['user'], computed: [{ relation: 'editor' }] },
						{ name: 'can_view', directTypes: [], computed: [{ relation: 'viewer' }] },
						{ name: 'can_edit', directTypes: [], computed: [{ relation: 'editor' }] },
						{ name: 'can_delete', directTypes: [], computed: [{ relation: 'owner' }] },
					],
				},
			],
		},
	},
	{
		name: 'Folders with inheritance',
		description: 'Documents inherit viewers from their parent folder.',
		model: {
			types: [
				{ name: 'user', relations: [] },
				{
					name: 'folder',
					relations: [
						{ name: 'owner', directTypes: ['user'], computed: [] },
						{ name: 'viewer', directTypes: ['user'], computed: [{ relation: 'owner' }] },
					],
				},
				{
					name: 'document',
					relations: [
						{ name: 'parent', directTypes: ['folder'], computed: [] },
						{ name: 'owner', directTypes: ['user'], computed: [] },
						{
							name: 'viewer',
							directTypes: ['user'],
							computed: [{ relation: 'owner' }, { relation: 'viewer', from: 'parent' }],
						},
						{ name: 'can_view', directTypes: [], computed: [{ relation: 'viewer' }] },
					],
				},
			],
		},
	},
	{
		name: 'Org / Team / Project',
		description: 'Project access flows from team and organization membership.',
		model: {
			types: [
				{ name: 'user', relations: [] },
				{
					name: 'organization',
					relations: [{ name: 'member', directTypes: ['user'], computed: [] }],
				},
				{
					name: 'team',
					relations: [
						{ name: 'org', directTypes: ['organization'], computed: [] },
						{
							name: 'member',
							directTypes: ['user'],
							computed: [{ relation: 'member', from: 'org' }],
						},
					],
				},
				{
					name: 'project',
					relations: [
						{ name: 'team', directTypes: ['team'], computed: [] },
						{
							name: 'viewer',
							directTypes: ['user'],
							computed: [{ relation: 'member', from: 'team' }],
						},
						{ name: 'can_view', directTypes: [], computed: [{ relation: 'viewer' }] },
					],
				},
			],
		},
	},
];
