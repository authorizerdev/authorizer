import React, { useMemo, useState } from 'react';
import { Plus, X, Check, AlertCircle } from 'lucide-react';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Label } from '../../components/ui/label';
import {
	generateDsl,
	parseDsl,
	rbacModel,
	sanitizeRelationName,
	summarize,
	RESERVED_TYPES,
} from './modelDsl';

// RbacBuilder is the friendly default for Step 1: a roles × permissions matrix
// that most admins already have a mental model for. It never asks the user to
// read or write DSL — it generates a standard OpenFGA RBAC model and hands the
// finished DSL to the parent (which owns saving). The raw DSL stays available
// behind the "Advanced" mode for anyone who wants the full language.
const DEFAULT_PERMISSIONS = ['view', 'edit', 'delete'];

// A sensible starting grant so the matrix is never empty on first paint:
// admins get everything, editors can view + edit, viewers can view.
const defaultGrant = (
	roles: string[],
	perms: string[],
): Record<string, string[]> => {
	const grant: Record<string, string[]> = {};
	for (const role of roles) {
		if (/admin|owner/i.test(role)) grant[role] = [...perms];
		else if (/edit|writ|maintain/i.test(role))
			grant[role] = perms.filter((p) => p !== 'delete');
		else grant[role] = perms.filter((p) => /view|read/i.test(p));
	}
	return grant;
};

// TokenList renders an editable list of names (roles or permissions) as
// removable chips plus an "add" input. Names are sanitized to valid relation
// identifiers on add.
const TokenList = ({
	label,
	help,
	items,
	onAdd,
	onRemove,
	placeholder,
}: {
	label: string;
	help: string;
	items: string[];
	onAdd: (name: string) => void;
	onRemove: (name: string) => void;
	placeholder: string;
}) => {
	const [draft, setDraft] = useState('');
	const commit = () => {
		const name = sanitizeRelationName(draft);
		if (name) onAdd(name);
		setDraft('');
	};
	return (
		<div>
			<Label className="text-sm font-medium text-gray-700">{label}</Label>
			<p className="mb-2 mt-0.5 text-xs text-gray-500">{help}</p>
			<div className="flex flex-wrap items-center gap-2">
				{items.map((item) => (
					<span
						key={item}
						className="inline-flex items-center gap-1 rounded-full border border-gray-200 bg-gray-50 py-1 pl-3 pr-1 text-sm text-gray-800"
					>
						<span className="font-mono text-xs">{item}</span>
						<button
							type="button"
							onClick={() => onRemove(item)}
							aria-label={`Remove ${item}`}
							className="flex h-5 w-5 items-center justify-center rounded-full text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-700"
						>
							<X className="h-3 w-3" aria-hidden="true" />
						</button>
					</span>
				))}
				<span className="inline-flex items-center gap-1">
					<Input
						value={draft}
						onChange={(e) => setDraft(e.target.value)}
						onKeyDown={(e) => {
							if (e.key === 'Enter') {
								e.preventDefault();
								commit();
							}
						}}
						placeholder={placeholder}
						className="h-8 w-32 text-sm"
						aria-label={`Add ${label.toLowerCase()}`}
						spellCheck={false}
					/>
					<Button
						type="button"
						variant="ghost"
						size="sm"
						onClick={commit}
						disabled={!sanitizeRelationName(draft)}
					>
						<Plus className="h-4 w-4" aria-hidden="true" />
					</Button>
				</span>
			</div>
		</div>
	);
};

// SEED_ROLES is the default matrix: a standard RBAC starting point with
// sensible grants. Instance roles are offered as suggestions, not forced in —
// app roles (like "user") often make poor object-scoped FGA roles.
const SEED_ROLES = ['admin', 'editor', 'viewer'];

const RbacBuilder = ({
	suggestedRoles = [],
	onApply,
}: {
	// Roles configured on the instance, offered as one-click additions to the
	// matrix (a fallback source of role names — never the default seed).
	suggestedRoles?: string[];
	// Called with the generated DSL when the admin is happy with the matrix.
	onApply: (dsl: string) => void;
}) => {
	const seedRoles = SEED_ROLES;
	// One or more object types to protect. All share the same roles × permissions
	// matrix; for resources that need different shapes, use Advanced (DSL).
	const [resourceTypes, setResourceTypes] = useState<string[]>(['document']);
	const [roles, setRoles] = useState<string[]>(() =>
		seedRoles.map(sanitizeRelationName).filter(Boolean),
	);
	const [permissions, setPermissions] = useState<string[]>(DEFAULT_PERMISSIONS);
	const [grant, setGrant] = useState<Record<string, string[]>>(() =>
		defaultGrant(
			seedRoles.map(sanitizeRelationName).filter(Boolean),
			DEFAULT_PERMISSIONS,
		),
	);
	const [showDsl, setShowDsl] = useState(false);

	const toggle = (role: string, perm: string) => {
		setGrant((prev) => {
			const current = prev[role] || [];
			const next = current.includes(perm)
				? current.filter((p) => p !== perm)
				: [...current, perm];
			return { ...prev, [role]: next };
		});
	};

	const addResource = (name: string) =>
		setResourceTypes((prev) => (prev.includes(name) ? prev : [...prev, name]));
	const removeResource = (name: string) =>
		setResourceTypes((prev) => prev.filter((r) => r !== name));
	const addRole = (name: string) =>
		setRoles((prev) => (prev.includes(name) ? prev : [...prev, name]));
	const removeRole = (name: string) => {
		setRoles((prev) => prev.filter((r) => r !== name));
		setGrant((prev) => {
			const next = { ...prev };
			delete next[name];
			return next;
		});
	};
	const addPermission = (name: string) =>
		setPermissions((prev) => (prev.includes(name) ? prev : [...prev, name]));
	const removePermission = (name: string) => {
		setPermissions((prev) => prev.filter((p) => p !== name));
		setGrant((prev) => {
			const next: Record<string, string[]> = {};
			for (const [role, perms] of Object.entries(prev)) {
				next[role] = perms.filter((p) => p !== name);
			}
			return next;
		});
	};

	// Sanitized resource names split into usable vs reserved (user/role).
	const cleanResources = resourceTypes
		.map(sanitizeRelationName)
		.filter(Boolean);
	const reservedResources = cleanResources.filter((r) =>
		RESERVED_TYPES.includes(r),
	);
	const validResources = cleanResources.filter(
		(r) => !RESERVED_TYPES.includes(r),
	);
	// A label for the matrix caption — the first protected resource, or generic.
	const resourceLabel = validResources[0] || 'resource';

	const model = useMemo(
		() => rbacModel({ resourceTypes, roles, permissions, grant }),
		[resourceTypes, roles, permissions, grant],
	);
	const dsl = model ? generateDsl(model).trimEnd() : '';
	const summary = useMemo(() => {
		if (!dsl) return [];
		const parsed = parseDsl(dsl);
		return parsed.supported && parsed.model ? summarize(parsed.model) : [];
	}, [dsl]);

	// A precise reason the matrix can't be turned into a model yet, so the
	// disabled state is never a dead end.
	const blocker = !validResources.length
		? reservedResources.length
			? `"${reservedResources[0]}" is reserved — add a different resource type.`
			: 'Add a resource type to protect (e.g. document).'
		: !roles.length
			? 'Add at least one role.'
			: !model
				? 'Grant at least one permission to a role (tick a box below).'
				: '';

	return (
		<div className="space-y-6">
			<div>
				<TokenList
					label="What are you protecting?"
					help="The object types access is granted on — e.g. document, project, report. Add as many as you like; they share the roles & permissions below."
					items={resourceTypes}
					onAdd={addResource}
					onRemove={removeResource}
					placeholder="resource type"
				/>
				{reservedResources.length > 0 && (
					<p className="mt-1 flex items-center gap-1 text-xs text-red-600">
						<AlertCircle className="h-3.5 w-3.5" aria-hidden="true" />
						&ldquo;{reservedResources[0]}&rdquo; is reserved — try{' '}
						<code className="rounded bg-red-50 px-1">
							{reservedResources[0]}_item
						</code>
						.
					</p>
				)}
			</div>

			<div>
				<TokenList
					label="Roles"
					help="Named sets of access you assign to people — e.g. admin, editor, viewer."
					items={roles}
					onAdd={addRole}
					onRemove={removeRole}
					placeholder="role name"
				/>
				{/* Instance roles as optional one-click additions (never forced). */}
				{(() => {
					const suggestions = suggestedRoles
						.map(sanitizeRelationName)
						.filter((r) => r && !roles.includes(r));
					if (!suggestions.length) return null;
					return (
						<div className="mt-2 flex flex-wrap items-center gap-1.5">
							<span className="text-xs text-gray-400">
								From your instance config:
							</span>
							{suggestions.map((r) => (
								<button
									key={r}
									type="button"
									onClick={() => addRole(r)}
									className="inline-flex items-center gap-1 rounded-full border border-dashed border-gray-300 px-2.5 py-0.5 font-mono text-xs text-gray-500 transition-colors hover:border-blue-300 hover:text-blue-600"
								>
									<Plus className="h-3 w-3" aria-hidden="true" />
									{r}
								</button>
							))}
						</div>
					);
				})()}
			</div>

			<TokenList
				label="Permissions"
				help="The actions people can take on the resource — e.g. view, edit, delete."
				items={permissions}
				onAdd={addPermission}
				onRemove={removePermission}
				placeholder="permission"
			/>

			{/* Roles × permissions matrix */}
			<div>
				<Label className="text-sm font-medium text-gray-700">
					Which roles can do what?
				</Label>
				<p className="mb-2 mt-0.5 text-xs text-gray-500">
					Tick a box to let a role perform an action on a {resourceLabel}
					{validResources.length > 1 ? ' (and your other resources)' : ''}.
				</p>
				{roles.length && permissions.length ? (
					<div className="overflow-x-auto rounded-lg border border-gray-200">
						<table className="w-full border-collapse text-sm">
							<thead>
								<tr className="border-b border-gray-200 bg-gray-50">
									<th className="px-4 py-2.5 text-left font-medium text-gray-600">
										Role
									</th>
									{permissions.map((perm) => (
										<th
											key={perm}
											className="px-4 py-2.5 text-center font-medium text-gray-600"
										>
											<span className="font-mono text-xs">can_{perm}</span>
										</th>
									))}
								</tr>
							</thead>
							<tbody>
								{roles.map((role) => (
									<tr
										key={role}
										className="border-b border-gray-100 last:border-0"
									>
										<td className="px-4 py-2.5 font-mono text-xs text-gray-800">
											{role}
										</td>
										{permissions.map((perm) => {
											const checked = (grant[role] || []).includes(perm);
											return (
												<td key={perm} className="px-4 py-2.5 text-center">
													<label className="inline-flex cursor-pointer items-center justify-center">
														<input
															type="checkbox"
															checked={checked}
															onChange={() => toggle(role, perm)}
															className="h-4 w-4 cursor-pointer rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-400"
															aria-label={`${role} can ${perm} ${resourceLabel}`}
														/>
													</label>
												</td>
											);
										})}
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<p className="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-400">
						Add at least one role and one permission to build the matrix.
					</p>
				)}
			</div>

			{/* Live "in plain English" so the matrix is never a black box */}
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

			{/* Optional DSL peek for the curious — collapsed by default */}
			{dsl && (
				<div>
					<button
						type="button"
						onClick={() => setShowDsl((v) => !v)}
						className="text-xs font-medium text-blue-600 hover:text-blue-700"
					>
						{showDsl ? 'Hide' : 'Show'} the generated model (OpenFGA DSL)
					</button>
					{showDsl && (
						<pre className="mt-2 overflow-x-auto rounded-lg border border-gray-200 bg-gray-900 p-3 font-mono text-xs leading-relaxed text-gray-100">
							{dsl}
						</pre>
					)}
				</div>
			)}

			<div className="flex flex-wrap items-center gap-3 border-t border-gray-100 pt-4">
				<Button type="button" onClick={() => onApply(dsl)} disabled={!model}>
					<Check className="mr-2 h-4 w-4" aria-hidden="true" />
					Use this model
				</Button>
				{blocker && <span className="text-xs text-gray-500">{blocker}</span>}
			</div>
		</div>
	);
};

export default RbacBuilder;
