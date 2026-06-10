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
	for (const c of r.computed)
		parts.push(c.from ? `${c.relation} from ${c.from}` : c.relation);
	return parts.join(' or ');
}

export function generateDsl(model: ModelDraft): string {
	const lines: string[] = ['model', '  schema 1.1', ''];
	for (const t of model.types) {
		lines.push(`type ${t.name}`);
		if (t.relations.length) {
			lines.push('  relations');
			for (const r of t.relations)
				lines.push(`    define ${r.name}: ${relationExpr(r)}`);
		}
	}
	return lines.join('\n') + '\n';
}

// parseDsl best-effort parses the simple subset (direct + union + inheritance)
// so we can render a plain-English summary. Models with advanced constructs
// return supported=false (no summary shown).
export function parseDsl(dsl: string): {
	model: ModelDraft | null;
	supported: boolean;
} {
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
			if (/\bbut\s+not\b|\band\b|[()]|\bwith\b/.test(expr))
				return { model: null, supported: false };
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
			if (r.directTypes.length)
				bits.push(`assigned (${r.directTypes.join(', ')})`);
			for (const c of r.computed)
				bits.push(
					c.from ? `${c.relation} of its ${c.from}` : `a ${c.relation}`,
				);
			out.push(
				`A user who is ${bits.join(' OR ')} is "${r.name}" of a ${t.name}.`,
			);
		}
	}
	return out;
}

export function sanitizeRelationName(role: string): string {
	return role
		.trim()
		.replace(/[^a-zA-Z0-9_]/g, '_')
		.replace(/^_+|_+$/g, '');
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
	rel.push({
		name: 'can_access',
		directTypes: [],
		computed: rel.map((r) => ({ relation: r.name })),
	});
	return {
		types: [
			{ name: 'user', relations: [] },
			{ name: 'resource', relations: rel },
		],
	};
}

// dedupeNames sanitizes a list of role/permission names and drops blanks and
// duplicates while preserving order.
function dedupeNames(names: string[]): string[] {
	const seen = new Set<string>();
	const out: string[] = [];
	for (const raw of names) {
		const name = sanitizeRelationName(raw);
		if (!name || seen.has(name)) continue;
		seen.add(name);
		out.push(name);
	}
	return out;
}

// RbacConfig describes a roles × permissions matrix applied to one or more
// resource types: a set of roles, a set of actions, and which roles are granted
// which actions. Every listed resource type shares the same matrix.
export interface RbacConfig {
	// The object types being protected (e.g. ["document", "project"]). Each
	// becomes its own type with the same role/permission relations.
	resourceTypes: string[];
	roles: string[];
	permissions: string[];
	// grant[role] = list of permission names that role is allowed.
	grant: Record<string, string[]>;
}

// RESERVED_TYPES are the type names the RBAC builder always emits; a resource
// type may not reuse them.
export const RESERVED_TYPES = ['user', 'role'];

// rbacModel turns a roles × permissions matrix into a standard OpenFGA RBAC
// model with one type per protected resource:
//
//   type user
//   type role
//     relations
//       define assignee: [user]
//   type <resource>            (one per resourceTypes entry)
//     relations
//       define <role>: [user, role#assignee]   (one per role)
//       define can_<perm>: <roles-with-perm>    (one per granted action)
//
// Each role relation accepts a direct user OR a whole role userset, so admins
// can grant a single user or an entire role on an object. Returns null when the
// matrix is empty (no valid resource, no role, or no granted permission).
export function rbacModel(config: RbacConfig): ModelDraft | null {
	const resources = dedupeNames(config.resourceTypes).filter(
		(r) => !RESERVED_TYPES.includes(r),
	);
	if (!resources.length) return null;

	const roles = dedupeNames(config.roles);
	const perms = dedupeNames(config.permissions);
	if (!roles.length) return null;

	const relations: RelationDef[] = [];
	for (const role of roles) {
		relations.push({
			name: role,
			directTypes: ['user', 'role#assignee'],
			computed: [],
		});
	}

	let grantedAny = false;
	for (const perm of perms) {
		const rolesWithPerm = roles.filter((r) =>
			(config.grant[r] || []).map(sanitizeRelationName).includes(perm),
		);
		if (!rolesWithPerm.length) continue;
		grantedAny = true;
		relations.push({
			name: `can_${perm}`,
			directTypes: [],
			computed: rolesWithPerm.map((r) => ({ relation: r })),
		});
	}
	if (!grantedAny) return null;

	// Each resource gets its own copy of the relation set (cloned so the
	// generated types never share mutable references).
	const resourceTypes: TypeDef[] = resources.map((name) => ({
		name,
		relations: relations.map((r) => ({
			name: r.name,
			directTypes: [...r.directTypes],
			computed: r.computed.map((c) => ({ ...c })),
		})),
	}));

	return {
		types: [
			{ name: 'user', relations: [] },
			{
				name: 'role',
				relations: [{ name: 'assignee', directTypes: ['user'], computed: [] }],
			},
			...resourceTypes,
		],
	};
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
		name: 'Org → project → resource',
		description:
			'Grant once on the org; every project and resource under it inherits. Add a direct grant for a single resource exception.',
		dsl: `model
  schema 1.1

type user

type organization
  relations
    define admin: [user]
    define editor: [user] or admin
    define viewer: [user] or editor
    define can_view: viewer
    define can_edit: editor

type project
  relations
    define org: [organization]
    define editor: [user] or editor from org
    define viewer: [user] or editor or viewer from org
    define can_view: viewer
    define can_edit: editor

type resource
  relations
    define project: [project]
    define editor: [user] or editor from project
    define viewer: [user] or editor or viewer from project
    define can_view: viewer
    define can_edit: editor`,
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
		name: 'Company roles (RBAC)',
		description:
			'Job roles — labour, manager, executive — with escalating permissions on a record.',
		dsl: `model
  schema 1.1

type user

type role
  relations
    define assignee: [user]

type record
  relations
    define labour: [user, role#assignee]
    define manager: [user, role#assignee]
    define executive: [user, role#assignee]
    define can_delete: executive
    define can_approve: executive
    define can_edit: manager or can_delete
    define can_view: labour or can_edit`,
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
