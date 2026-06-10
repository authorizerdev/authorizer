import React, { useState } from 'react';
import { useClient } from 'urql';
import { Search, ShieldCheck } from 'lucide-react';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
} from './ui/dialog';
import { ListPermissionsQuery } from '../graphql/queries';
import { isFgaNotEnabledError } from '../lib/utils';
import type { ListPermissionsResponse, User } from '../types';

interface UserPermissionsModalProps {
	user: User | null;
	open: boolean;
	onClose: () => void;
}

// UserPermissionsModal lets an admin answer "which objects does this user hold
// a permission on?" straight from the Users table. It calls the public
// list_permissions API with an explicit subject — honored because the
// dashboard session is super-admin.
const UserPermissionsModal = ({
	user,
	open,
	onClose,
}: UserPermissionsModalProps) => {
	const client = useClient();
	const [relation, setRelation] = useState('');
	const [objectType, setObjectType] = useState('');
	const [objects, setObjects] = useState<string[] | null>(null);
	const [error, setError] = useState('');
	const [running, setRunning] = useState(false);

	if (!user) return null;

	const handleList = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!relation.trim() || !objectType.trim()) {
			setError('relation and object type are both required');
			return;
		}
		setRunning(true);
		setError('');
		setObjects(null);
		try {
			const res = await client
				.query<ListPermissionsResponse>(
					ListPermissionsQuery,
					{
						params: {
							relation: relation.trim(),
							object_type: objectType.trim(),
							user: `user:${user.id}`,
						},
					},
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
			setObjects(res.data?.list_permissions?.objects ?? []);
		} catch {
			setError('Failed to list permissions');
		} finally {
			setRunning(false);
		}
	};

	const close = () => {
		setObjects(null);
		setError('');
		onClose();
	};

	return (
		<Dialog open={open} onOpenChange={(o) => !o && close()}>
			<DialogContent className="max-w-lg">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<ShieldCheck className="h-5 w-5 text-blue-600" aria-hidden="true" />
						Permissions · {user.email || user.phone_number || user.id}
					</DialogTitle>
					<DialogDescription>
						List the objects{' '}
						<code className="rounded bg-gray-100 px-1 py-0.5 text-xs">
							user:{user.id}
						</code>{' '}
						holds a permission on. Pick the permission (relation) and object
						type from your authorization model.
					</DialogDescription>
				</DialogHeader>
				<form onSubmit={handleList} className="space-y-3">
					<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
						<div className="space-y-1">
							<Label htmlFor="perm-relation">Permission (relation)</Label>
							<Input
								id="perm-relation"
								placeholder="can_view"
								value={relation}
								onChange={(e) => setRelation(e.target.value)}
								spellCheck={false}
							/>
						</div>
						<div className="space-y-1">
							<Label htmlFor="perm-object-type">Object type</Label>
							<Input
								id="perm-object-type"
								placeholder="document"
								value={objectType}
								onChange={(e) => setObjectType(e.target.value)}
								spellCheck={false}
							/>
						</div>
					</div>
					<Button type="submit" disabled={running}>
						<Search className="mr-2 h-4 w-4" aria-hidden="true" />
						{running ? 'Listing…' : 'List permissions'}
					</Button>
				</form>
				{error && (
					<p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
						{error}
					</p>
				)}
				{objects !== null &&
					(objects.length ? (
						<div className="max-h-64 overflow-y-auto rounded-lg border border-gray-200">
							<ul className="divide-y divide-gray-100">
								{objects.map((o) => (
									<li
										key={o}
										className="px-3 py-2 font-mono text-xs text-gray-800"
									>
										{o}
									</li>
								))}
							</ul>
						</div>
					) : (
						<p className="rounded-lg border border-dashed border-gray-200 p-3 text-sm text-gray-500">
							No <code className="font-mono text-xs">{objectType}</code> objects
							with <code className="font-mono text-xs">{relation}</code> for
							this user.
						</p>
					))}
			</DialogContent>
		</Dialog>
	);
};

export default UserPermissionsModal;
