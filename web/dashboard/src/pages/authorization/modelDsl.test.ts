import { describe, expect, it } from 'vitest';
import {
	generateDsl,
	parseDsl,
	summarize,
	rolesTemplate,
	sanitizeRelationName,
	rbacModel,
	MODEL_EXAMPLES,
	RESERVED_TYPES,
	type RbacConfig,
} from './modelDsl';

describe('sanitizeRelationName', () => {
	it('keeps valid identifiers untouched', () => {
		expect(sanitizeRelationName('viewer')).toBe('viewer');
		expect(sanitizeRelationName('can_edit')).toBe('can_edit');
	});

	it('replaces invalid characters with underscores and trims edges', () => {
		expect(sanitizeRelationName('  power user ')).toBe('power_user');
		expect(sanitizeRelationName('read-only')).toBe('read_only');
		expect(sanitizeRelationName('admin!')).toBe('admin');
	});

	it('returns empty string for names with no usable characters', () => {
		expect(sanitizeRelationName('   ')).toBe('');
		expect(sanitizeRelationName('!!!')).toBe('');
	});
});

describe('generateDsl / parseDsl round-trip', () => {
	it('serializes a model and parses it back to an equivalent shape', () => {
		const dsl = generateDsl({
			types: [
				{ name: 'user', relations: [] },
				{
					name: 'document',
					relations: [
						{ name: 'viewer', directTypes: ['user'], computed: [] },
						{ name: 'editor', directTypes: ['user'], computed: [] },
						{
							name: 'can_view',
							directTypes: [],
							computed: [{ relation: 'viewer' }, { relation: 'editor' }],
						},
					],
				},
			],
		});
		expect(dsl).toContain('type user');
		expect(dsl).toContain('define viewer: [user]');
		expect(dsl).toContain('define can_view: viewer or editor');

		const parsed = parseDsl(dsl);
		expect(parsed.supported).toBe(true);
		expect(parsed.model?.types.map((t) => t.name)).toEqual([
			'user',
			'document',
		]);
		const doc = parsed.model?.types.find((t) => t.name === 'document');
		expect(doc?.relations.find((r) => r.name === 'can_view')?.computed).toEqual(
			[{ relation: 'viewer' }, { relation: 'editor' }],
		);
	});

	it('parses inheritance ("from") relations', () => {
		const parsed = parseDsl(`model
  schema 1.1

type folder
  relations
    define viewer: [user]

type document
  relations
    define parent: [folder]
    define viewer: [user] or viewer from parent`);
		expect(parsed.supported).toBe(true);
		const doc = parsed.model?.types.find((t) => t.name === 'document');
		const viewer = doc?.relations.find((r) => r.name === 'viewer');
		expect(viewer?.directTypes).toEqual(['user']);
		expect(viewer?.computed).toEqual([{ relation: 'viewer', from: 'parent' }]);
	});

	it('flags advanced constructs as unsupported (no plain-English summary)', () => {
		expect(
			parseDsl('type x\n  relations\n    define a: [user] but not b').supported,
		).toBe(false);
		expect(
			parseDsl('type x\n  relations\n    define a: b and c').supported,
		).toBe(false);
		expect(
			parseDsl('type x\n  relations\n    define a: (b or c)').supported,
		).toBe(false);
		expect(
			parseDsl('type x\n  relations\n    define a: [user with cond]').supported,
		).toBe(false);
	});

	it('treats an empty or comment-only document as unsupported', () => {
		expect(parseDsl('').supported).toBe(false);
		expect(parseDsl('model\n  schema 1.1\n# just a comment').supported).toBe(
			false,
		);
	});
});

describe('summarize', () => {
	it('describes direct and computed relations in plain English', () => {
		const lines = summarize({
			types: [
				{
					name: 'document',
					relations: [
						{ name: 'viewer', directTypes: ['user'], computed: [] },
						{
							name: 'can_view',
							directTypes: [],
							computed: [
								{ relation: 'viewer' },
								{ relation: 'editor', from: 'parent' },
							],
						},
					],
				},
			],
		});
		expect(lines[0]).toContain('assigned (user)');
		expect(lines[0]).toContain('"viewer" of a document');
		expect(lines[1]).toContain('editor of its parent');
	});
});

describe('rolesTemplate', () => {
	it('builds a resource model with one relation per role plus can_access', () => {
		const model = rolesTemplate(['admin', 'user']);
		expect(model).not.toBeNull();
		const resource = model?.types.find((t) => t.name === 'resource');
		expect(resource?.relations.map((r) => r.name)).toEqual([
			'admin',
			'user',
			'can_access',
		]);
		const canAccess = resource?.relations.find((r) => r.name === 'can_access');
		expect(canAccess?.computed).toEqual([
			{ relation: 'admin' },
			{ relation: 'user' },
		]);
	});

	it('dedupes and sanitizes role names, returns null when nothing usable', () => {
		// "power user" and "power_user" both sanitize to the same name; the
		// blank is dropped. Dedup is case-sensitive (OpenFGA relations are).
		const model = rolesTemplate(['power user', 'power_user', '   ']);
		const resource = model?.types.find((t) => t.name === 'resource');
		expect(resource?.relations.map((r) => r.name)).toEqual([
			'power_user',
			'can_access',
		]);
		expect(rolesTemplate([])).toBeNull();
		expect(rolesTemplate(['   ', '!!!'])).toBeNull();
	});
});

describe('rbacModel', () => {
	const base: RbacConfig = {
		resourceTypes: ['document'],
		roles: ['admin', 'editor', 'viewer'],
		permissions: ['view', 'edit', 'delete'],
		grant: {
			admin: ['view', 'edit', 'delete'],
			editor: ['view', 'edit'],
			viewer: ['view'],
		},
	};

	it('produces user, role and resource types', () => {
		const model = rbacModel(base);
		expect(model?.types.map((t) => t.name)).toEqual([
			'user',
			'role',
			'document',
		]);
		const role = model?.types.find((t) => t.name === 'role');
		expect(role?.relations).toEqual([
			{ name: 'assignee', directTypes: ['user'], computed: [] },
		]);
	});

	it('emits a role relation accepting a direct user or a role userset', () => {
		const dsl = generateDsl(rbacModel(base)!);
		expect(dsl).toContain('define admin: [user, role#assignee]');
		expect(dsl).toContain('define editor: [user, role#assignee]');
	});

	it('emits can_<perm> as the union of roles granted that permission', () => {
		const dsl = generateDsl(rbacModel(base)!);
		expect(dsl).toContain('define can_view: admin or editor or viewer');
		expect(dsl).toContain('define can_edit: admin or editor');
		expect(dsl).toContain('define can_delete: admin');
	});

	it('skips permissions that no role is granted', () => {
		const model = rbacModel({
			...base,
			permissions: ['view', 'archive'],
			grant: { admin: ['view'], editor: ['view'], viewer: ['view'] },
		});
		const resource = model?.types.find((t) => t.name === 'document');
		expect(
			resource?.relations.find((r) => r.name === 'can_archive'),
		).toBeUndefined();
		expect(
			resource?.relations.find((r) => r.name === 'can_view'),
		).toBeDefined();
	});

	it('emits one type per resource, all sharing the same matrix', () => {
		const dsl = generateDsl(
			rbacModel({ ...base, resourceTypes: ['document', 'project'] })!,
		);
		expect(dsl).toContain('type document');
		expect(dsl).toContain('type project');
		// Both resources get the same permission relations.
		expect(
			dsl.match(/define can_view: admin or editor or viewer/g),
		).toHaveLength(2);
		expect(dsl.match(/define admin: \[user, role#assignee\]/g)).toHaveLength(2);
	});

	it('dedupes resources and drops reserved ones while keeping valid ones', () => {
		const model = rbacModel({
			...base,
			resourceTypes: ['document', 'document', 'user', 'role', 'project'],
		});
		expect(model?.types.map((t) => t.name)).toEqual([
			'user',
			'role',
			'document',
			'project',
		]);
	});

	it('sanitizes resource, role and permission names', () => {
		const dsl = generateDsl(
			rbacModel({
				resourceTypes: ['Customer Record'],
				roles: ['Account Owner'],
				permissions: ['Full Access'],
				grant: { Account_Owner: ['Full_Access'] },
			})!,
		);
		expect(dsl).toContain('type Customer_Record');
		expect(dsl).toContain('define Account_Owner: [user, role#assignee]');
		expect(dsl).toContain('define can_Full_Access: Account_Owner');
	});

	it('returns null for empty, reserved, or fully-ungranted matrices', () => {
		expect(rbacModel({ ...base, roles: [] })).toBeNull();
		expect(rbacModel({ ...base, resourceTypes: [] })).toBeNull();
		expect(rbacModel({ ...base, resourceTypes: [''] })).toBeNull();
		// A matrix whose only resources are reserved types yields no model.
		expect(rbacModel({ ...base, resourceTypes: RESERVED_TYPES })).toBeNull();
		expect(
			rbacModel({ ...base, grant: { admin: [], editor: [], viewer: [] } }),
		).toBeNull();
	});

	it('always generates a model that parses back to a supported summary', () => {
		const dsl = generateDsl(rbacModel(base)!);
		const parsed = parseDsl(dsl);
		expect(parsed.supported).toBe(true);
		expect(summarize(parsed.model!).length).toBeGreaterThan(0);
	});
});

describe('MODEL_EXAMPLES catalog', () => {
	it('every example has a name, description and non-empty DSL starting with "model"', () => {
		expect(MODEL_EXAMPLES.length).toBeGreaterThan(0);
		for (const ex of MODEL_EXAMPLES) {
			expect(ex.name).toBeTruthy();
			expect(ex.description).toBeTruthy();
			expect(ex.dsl.trim().startsWith('model')).toBe(true);
			// parseDsl must never throw on a shipped example (advanced ones may be
			// flagged unsupported, but they must still parse cleanly).
			expect(() => parseDsl(ex.dsl)).not.toThrow();
		}
	});

	it('includes a relatable company-roles RBAC example that summarizes', () => {
		const ex = MODEL_EXAMPLES.find((e) => e.name === 'Company roles (RBAC)');
		expect(ex).toBeDefined();
		const parsed = parseDsl(ex!.dsl);
		expect(parsed.supported).toBe(true);
		const lines = summarize(parsed.model!);
		// labour / manager / executive all appear as record relations.
		expect(lines.join(' ')).toContain('labour');
		expect(lines.join(' ')).toContain('executive');
	});

	it('includes an org → project → resource hierarchy example with inheritance', () => {
		const ex = MODEL_EXAMPLES.find(
			(e) => e.name === 'Org → project → resource',
		);
		expect(ex).toBeDefined();
		const parsed = parseDsl(ex!.dsl);
		expect(parsed.supported).toBe(true);
		// resource.viewer inherits from project (the grant-once-inherit chain).
		const resource = parsed.model!.types.find((t) => t.name === 'resource');
		const viewer = resource?.relations.find((r) => r.name === 'viewer');
		expect(viewer?.computed).toContainEqual({
			relation: 'viewer',
			from: 'project',
		});
	});
});
