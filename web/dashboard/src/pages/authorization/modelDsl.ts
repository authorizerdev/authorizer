// modelDsl.ts — helpers for the authorization-model editor: a plain-English
// summary of a (simple) model, a roles-derived example, and a catalog of
// ready-to-use OpenFGA model examples (raw DSL, so they can use the full
// language — usersets, exclusions, conditions).

export interface ComputedTerm {
	relation: string;
	from?: string;
}
export interface RelationDef {
	name: string;
	directTypes: string[];
	computed: ComputedTerm[];
}
export interface TypeDef {
	name: string;
	relations: RelationDef[];
}
export interface ModelDraft {
	types: TypeDef[];
}

const IDENT = /^[a-zA-Z0-9_]+$/;

function relationExpr(r: RelationDef): string {
	const parts: string[] = [];
	if (r.directTypes.length) parts.push(`[${r.directTypes.join(', ')}]`);
	for (const c of r.computed) parts.push(c.from ? `${c.relation} from ${c.from}` : c.relation);
	return parts.join(' or ');
}

export function generateDsl(model: ModelDraft): string {
	const lines: string[] = ['model', '  schema 1.1', ''];
	for (const t of model.types) {
		lines.push(`type ${t.name}`);
		if (t.relations.length) {
			lines.push('  relations');
			for (const r of t.relations) lines.push(`    define ${r.name}: ${relationExpr(r)}`);
		}
	}
	return lines.join('\n') + '\n';
}

// parseDsl best-effort parses the simple subset (direct + union + inheritance)
// so we can render a plain-English summary. Models with advanced constructs
// return supported=false (no summary shown).
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
			if (/\bbut\s+not\b|\band\b|[()]|\bwith\b/.test(expr)) return { model: null, supported: false };
			const rel: RelationDef = { name: m[1], directTypes: [], computed: [] };
			for (const partRaw of expr.split(/\s+or\s+/)) {
				const part = partRaw.trim();
				if (!part) continue;
				if (part.startsWith('[') && part.endsWith(']')) {
					rel.directTypes.push(...part.slice(1, -1).split(',').map((s) => s.trim()).filter(Boolean));
				} else if (/\sfrom\s/.test(part)) {
					const fm = part.match(/^([a-zA-Z0-9_]+)\s+from\s+([a-zA-Z0-9_]+)$/);
					if (!fm) return { model: null, supported: false };
					rel.computed.push({ relation: fm[1], from: fm[2] });
				} else if (IDENT.test(part)) {
					rel.computed.push({ relation: part });
				} else {
					return { model: null, supported: false };
				}
			}
			current.relations.push(rel);
			continue;
		}
		return { model: null, supported: false };
	}
	if (!types.length) return { model: null, supported: false };
	return { model: { types }, supported: true };
}

export function summarize(model: ModelDraft): string[] {
	const out: string[] = [];
	for (const t of model.types) {
		for (const r of t.relations) {
			const bits: string[] = [];
			if (r.directTypes.length) bits.push(`assigned (${r.directTypes.join(', ')})`);
			for (const c of r.computed) bits.push(c.from ? `${c.relation} of its ${c.from}` : `a ${c.relation}`);
			out.push(`A user who is ${bits.join(' OR ')} is "${r.name}" of a ${t.name}.`);
		}
	}
	return out;
}

export function sanitizeRelationName(role: string): string {
	return role.trim().replace(/[^a-zA-Z0-9_]/g, '_').replace(/^_+|_+$/g, '');
}

// rolesTemplate builds a model from the instance's configured roles: each role
// is a relation on a `resource`, plus a `can_access` permission.
export function rolesTemplate(roles: string[]): ModelDraft | null {
	const seen = new Set<string>();
	const rel: RelationDef[] = [];
	for (const raw of roles) {
		const name = sanitizeRelationName(raw);
		if (!name || seen.has(name)) continue;
		seen.add(name);
		rel.push({ name, directTypes: ['user'], computed: [] });
	}
	if (!rel.length) return null;
	rel.push({ name: 'can_access', directTypes: [], computed: rel.map((r) => ({ relation: r.name })) });
	return { types: [{ name: 'user', relations: [] }, { name: 'resource', relations: rel }] };
}

// MODEL_EXAMPLES is a catalog of common authorization patterns as raw DSL.
export interface ModelExample {
	name: string;
	description: string;
	dsl: string;
}

export const MODEL_EXAMPLES: ModelExample[] = [
	{
		name: 'Document sharing',
		description: 'Owner → editor → viewer, with cascading permissions.',
		dsl: `model
  schema 1.1

type user

type document
  relations
    define owner: [user]
    define editor: [user] or owner
    define viewer: [user] or editor
    define can_view: viewer
    define can_edit: editor
    define can_delete: owner`,
	},
	{
		name: 'Folder hierarchy',
		description: 'Documents inherit viewers from their parent folder.',
		dsl: `model
  schema 1.1

type user

type folder
  relations
    define owner: [user]
    define viewer: [user] or owner

type document
  relations
    define parent: [folder]
    define owner: [user]
    define viewer: [user] or owner or viewer from parent
    define can_view: viewer
    define can_edit: owner`,
	},
	{
		name: 'Organizations & teams',
		description: 'Team membership flows from organization membership.',
		dsl: `model
  schema 1.1

type user

type organization
  relations
    define admin: [user]
    define member: [user] or admin

type team
  relations
    define org: [organization]
    define member: [user] or member from org`,
	},
	{
		name: 'RBAC roles',
		description: 'Global roles assigned to users, referenced by resources.',
		dsl: `model
  schema 1.1

type user

type role
  relations
    define assignee: [user]

type resource
  relations
    define admin: [role#assignee]
    define editor: [user, role#assignee] or admin
    define viewer: [user, role#assignee] or editor
    define can_view: viewer
    define can_edit: editor
    define can_admin: admin`,
	},
	{
		name: 'Groups',
		description: 'Nestable user groups; grant access to a whole group.',
		dsl: `model
  schema 1.1

type user

type group
  relations
    define member: [user, group#member]

type document
  relations
    define viewer: [user, group#member]
    define can_view: viewer`,
	},
	{
		name: 'Block list (exclusion)',
		description: 'Everyone with viewer access, except blocked users.',
		dsl: `model
  schema 1.1

type user

type document
  relations
    define viewer: [user]
    define blocked: [user]
    define can_view: viewer but not blocked`,
	},
	{
		name: 'Multi-tenant SaaS',
		description: 'Organization → workspace → resource access flow.',
		dsl: `model
  schema 1.1

type user

type organization
  relations
    define member: [user]

type workspace
  relations
    define org: [organization]
    define admin: [user]
    define member: [user] or admin or member from org

type resource
  relations
    define workspace: [workspace]
    define editor: [user] or admin from workspace
    define viewer: [user] or editor or member from workspace
    define can_view: viewer
    define can_edit: editor`,
	},
	{
		name: 'GitHub-style repos',
		description: 'Org → repo with admin / maintainer / writer / reader.',
		dsl: `model
  schema 1.1

type user

type organization
  relations
    define owner: [user]
    define member: [user] or owner

type repository
  relations
    define org: [organization]
    define admin: [user] or owner from org
    define maintainer: [user] or admin
    define writer: [user] or maintainer
    define reader: [user] or writer or member from org
    define can_read: reader
    define can_push: writer
    define can_admin: admin`,
	},
	{
		name: 'Time-bound access (conditions)',
		description: 'ABAC: a grant that is only valid until it expires.',
		dsl: `model
  schema 1.1

type user

type document
  relations
    define viewer: [user with non_expired_grant]
    define can_view: viewer

condition non_expired_grant(current_time: timestamp, grant_time: timestamp, grant_duration: duration) {
  current_time < grant_time + grant_duration
}`,
	},
];
