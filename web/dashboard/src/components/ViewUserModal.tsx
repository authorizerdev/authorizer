import React from 'react';
import dayjs from 'dayjs';
import { Badge } from './ui/badge';
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
} from './ui/dialog';
import type { User } from '../types';

interface ViewUserModalProps {
	user: User | null;
	open: boolean;
	onClose: () => void;
}

const ViewUserModal = ({ user, open, onClose }: ViewUserModalProps) => {
	if (!user) return null;

	const fields: { label: string; value: React.ReactNode }[] = [
		{ label: 'ID', value: <span className="font-mono text-xs">{user.id}</span> },
		{ label: 'Email', value: user.email || '—' },
		{
			label: 'Email Verified',
			value: (
				<Badge variant={user.email_verified ? 'success' : 'warning'}>
					{user.email_verified ? 'Yes' : 'No'}
				</Badge>
			),
		},
		{ label: 'Given Name', value: user.given_name || '—' },
		{ label: 'Family Name', value: user.family_name || '—' },
		{ label: 'Middle Name', value: user.middle_name || '—' },
		{ label: 'Nickname', value: user.nickname || '—' },
		{ label: 'Phone Number', value: user.phone_number || '—' },
		{
			label: 'Phone Verified',
			value: (
				<Badge variant={user.phone_number_verified ? 'success' : 'warning'}>
					{user.phone_number_verified ? 'Yes' : 'No'}
				</Badge>
			),
		},
		{ label: 'Gender', value: user.gender || '—' },
		{ label: 'Birthdate', value: user.birthdate || '—' },
		{ label: 'Picture', value: user.picture || '—' },
		{ label: 'Signup Methods', value: user.signup_methods || '—' },
		{
			label: 'Roles',
			value: (
				<div className="flex flex-wrap gap-1">
					{user.roles.map((role) => (
						<Badge key={role} variant="secondary">
							{role}
						</Badge>
					))}
				</div>
			),
		},
		{
			label: 'MFA',
			value: (
				<Badge variant={user.is_multi_factor_auth_enabled ? 'success' : 'destructive'}>
					{user.is_multi_factor_auth_enabled ? 'Enabled' : 'Disabled'}
				</Badge>
			),
		},
		{
			label: 'Access',
			value: (
				<Badge variant={user.revoked_timestamp ? 'destructive' : 'success'}>
					{user.revoked_timestamp ? 'Revoked' : 'Active'}
				</Badge>
			),
		},
		{
			label: 'Created At',
			value: dayjs(user.created_at * 1000).format('MMM DD, YYYY HH:mm:ss'),
		},
		{
			label: 'Updated At',
			value: user.updated_at
				? dayjs(user.updated_at * 1000).format('MMM DD, YYYY HH:mm:ss')
				: '—',
		},
	];

	return (
		<Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
			<DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle>User Details</DialogTitle>
				</DialogHeader>
				<div className="grid gap-3 py-2">
					{fields.map(({ label, value }) => (
						<div
							key={label}
							className="grid grid-cols-[140px_1fr] gap-2 items-start text-sm"
						>
							<span className="font-medium text-gray-500">{label}</span>
							<span className="text-gray-900 break-all">{value}</span>
						</div>
					))}
				</div>
			</DialogContent>
		</Dialog>
	);
};

export default ViewUserModal;
