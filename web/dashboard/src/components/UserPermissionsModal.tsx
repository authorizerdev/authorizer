import React, { useCallback, useEffect, useState } from 'react';
import { useClient } from 'urql';
import { RefreshCw, Search, ShieldCheck, TriangleAlert } from 'lucide-react';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Skeleton } from './ui/skeleton';
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
} from './ui/dialog';
import { ListPermissionsQuery } from '../graphql/queries';
import { isFgaNotEnabledError } from '../lib/utils';
import type { ListPermissionsResponse, Permission, User } from '../types';

interface UserPermissionsModalProps {
	user: User | null;
	open: boolean;
	onClose: () => void;
}

// UserPermissionsModal lets an admin answer "what can this user access?"
// straight from the Users table. It calls the public list_permissions API with
// an explicit subject — honored because the dashboard session is super-admin.
// The COMPLETE permission list loads automatically when the modal opens; the
// form only narrows it (relation and/or object type), and all state resets on
// close.
const UserPermissionsModal = ({
	user,
	open,
	onClose,
}: UserPermissionsModalProps) => {
	const client = useClient();
	const [relation, setRelation] = useState('');
	const [objectType, setObjectType] = useState('');
	const [permissions, setPermissions] = useState<Permission[] | null>(null);
	const [truncated, setTruncated] = useState(false);
	const [error, setError] = useState('');
	const [running, setRunning] = useState(false);

	const userId = user?.id ?? '';

	const fetchPermissions = useCallback(
		async (relationFilter: string, typeFilter: string) => {
			if (!userId) return;
			setRunning(true);
			setError('');
			setPermissions(null);
			setTruncated(false);
			try {
				// Omit empty filters so the server enumerates every matching
				// (type, relation) pair of the model.
				const params: Record<string, string> = { user: `user:${userId}` };
				if (relationFilter) params.relation = relationFilter;
				if (typeFilter) params.object_type = typeFilter;
				const res = await client
					.query<ListPermissionsResponse>(
						ListPermissionsQuery,
						{ params },
						{ requestPolicy: 'network-only' },
					)
					.toPromise();
				if (res.error) {
					setError(
						isFgaNotEnabledError(res.error)
							? 'Fine-grained authorization is not enabled on this instance.'
							: res.error.message.replace('[GraphQL] ', ''),
					);
					return;
				}
				setPermissions(res.data?.list_permissions?.permissions ?? []);
				setTruncated(res.data?.list_permissions?.truncated ?? false);
			} catch {
				setError('Failed to list permissions');
			} finally {
				setRunning(false);
			}
		},
		[client, userId],
	);

	// Load the complete list as soon as the modal opens — no filter or click
	// needed to see what the user can access.
	useEffect(() => {
		if (open && userId) {
			void fetchPermissions('', '');
		}
	}, [open, userId, fetchPermissions]);

	if (!user) return null;

	const handleFilter = async (e: React.FormEvent) => {
		e.preventDefault();
		await fetchPermissions(relation.trim(), objectType.trim());
	};

	const close = () => {
		// Reset everything so the next open starts fresh for any user.
		setRelation('');
		setObjectType('');
		setPermissions(null);
		setTruncated(false);
		setError('');
		onClose();
	};

	const hasFilters = Boolean(relation.trim() || objectType.trim());

	return (
		<Dialog open={open} onOpenChange={(o) => !o && close()}>
			<DialogContent className="max-w-lg">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<ShieldCheck className="h-5 w-5 text-blue-600" aria-hidden="true" />
						Permissions · {user.email || user.phone_number || user.id}
					</DialogTitle>
					<DialogDescription>
						Everything{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5 text-xs">
							user:{user.id}
						</code>{' '}
						can access. Narrow by permission (relation) and/or object type if
						the list is long.
					</DialogDescription>
				</DialogHeader>
				<form onSubmit={handleFilter} className="space-y-3">
					<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
						<div className="space-y-1">
							<Label htmlFor="perm-relation">Permission (optional)</Label>
							<Input
								id="perm-relation"
								placeholder="can_view"
								value={relation}
								onChange={(e) => setRelation(e.target.value)}
								spellCheck={false}
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="perm-object-type">Object type (optional)</Label>
							<Input
								id="perm-object-type"
								placeholder="document"
								value={objectType}
								onChange={(e) => setObjectType(e.target.value)}
								spellCheck={false}
							/>
						</div>
					</div>
					<Button type="submit" variant="outline" disabled={running}>
						{hasFilters ? (
							<Search className="mr-2 h-4 w-4" aria-hidden="true" />
						) : (
							<RefreshCw className="mr-2 h-4 w-4" aria-hidden="true" />
						)}
						{running ? 'Listing…' : hasFilters ? 'Apply filters' : 'Refresh'}
					</Button>
				</form>
				{error && (
					<p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
						{error}
					</p>
				)}
				{truncated && (
					<p className="flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700">
						<TriangleAlert className="h-4 w-4 shrink-0" aria-hidden="true" />
						Showing the first 1000 permissions — more exist. Narrow by
						permission or object type to see the rest.
					</p>
				)}
				{running && permissions === null && !error && (
					<div className="space-y-2" aria-label="Loading permissions">
						<Skeleton className="h-8 w-full" />
						<Skeleton className="h-8 w-full" />
						<Skeleton className="h-8 w-2/3" />
					</div>
				)}
				{permissions !== null &&
					(permissions.length ? (
						<div className="max-h-64 overflow-y-auto rounded-lg border border-gray-200">
							<table className="w-full text-left">
								<thead className="sticky top-0 bg-gray-50">
									<tr className="text-xs font-medium text-gray-500">
										<th className="px-3 py-2">Object</th>
										<th className="px-3 py-2">Permission</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-100">
									{permissions.map((p) => (
										<tr key={`${p.object}#${p.relation}`}>
											<td className="px-3 py-2 font-mono text-xs text-gray-800">
												{p.object}
											</td>
											<td className="px-3 py-2">
												<span className="rounded bg-blue-50 px-1.5 py-0.5 font-mono text-xs text-blue-700">
													{p.relation}
												</span>
											</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>
					) : (
						<p className="rounded-lg border border-dashed border-gray-200 p-3 text-sm text-gray-500">
							{hasFilters ? (
								<>
									No permissions matching{' '}
									{relation.trim() && (
										<code className="font-mono text-xs">{relation.trim()}</code>
									)}
									{relation.trim() && objectType.trim() && ' on '}
									{objectType.trim() && (
										<code className="font-mono text-xs">
											{objectType.trim()}
										</code>
									)}{' '}
									for this user.
								</>
							) : (
								<>This user holds no permissions in the authorization model.</>
							)}
						</p>
					))}
			</DialogContent>
		</Dialog>
	);
};

export default UserPermissionsModal;
