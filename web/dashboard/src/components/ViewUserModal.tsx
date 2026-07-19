import React from 'react';
import { useClient } from 'urql';
import dayjs from 'dayjs';
import { Badge } from './ui/badge';
import MfaStatus from './MfaStatus';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './ui/dialog';
import { UserOrganizationsQuery } from '../graphql/queries';
import type {
	User,
	UserOrganization,
	UserOrganizationsResponse,
} from '../types';

interface ViewUserModalProps {
	user: User | null;
	open: boolean;
	// mfaServiceEnabled is the org-wide _admin_meta
	// .is_multi_factor_auth_service_enabled signal, passed down from the Users
	// page so the MFA status can distinguish "server has no MFA" from "user
	// hasn't enrolled".
	mfaServiceEnabled: boolean;
	onClose: () => void;
}

const ViewUserModal = ({
	user,
	open,
	mfaServiceEnabled,
	onClose,
}: ViewUserModalProps) => {
	const client = useClient();
	const [orgs, setOrgs] = React.useState<UserOrganization[]>([]);
	const [orgLoading, setOrgLoading] = React.useState(false);
	// orgError degrades the Organizations section gracefully: if the query
	// fails we simply hide the section rather than break the whole modal.
	const [orgError, setOrgError] = React.useState(false);

	const userId = user?.id;

	React.useEffect(() => {
		if (!open || !userId) {
			return;
		}
		let cancelled = false;
		setOrgLoading(true);
		setOrgError(false);
		setOrgs([]);
		client
			.query<UserOrganizationsResponse>(UserOrganizationsQuery, {
				params: { user_id: userId },
			})
			.toPromise()
			.then(({ data, error }) => {
				if (cancelled) {
					return;
				}
				if (error || !data?._user_organizations) {
					setOrgError(true);
				} else {
					setOrgs(data._user_organizations.user_organizations || []);
				}
				setOrgLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [open, userId, client]);

	if (!user) return null;

	const fields: { label: string; value: React.ReactNode }[] = [
		{
			label: 'ID',
			value: <span className="font-mono text-xs">{user.id}</span>,
		},
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
				<MfaStatus
					mfaServiceEnabled={mfaServiceEnabled}
					enrolledMethods={user.enrolled_mfa_methods}
					isMfaEnabled={user.is_multi_factor_auth_enabled}
				/>
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
				{!orgError && (
					<div className="border-t pt-3">
						<h3 className="mb-2 text-sm font-semibold text-gray-900">
							Organizations
						</h3>
						{orgLoading ? (
							<p className="text-sm text-gray-400">Loading…</p>
						) : orgs.length > 0 ? (
							<div className="grid gap-2">
								{orgs.map(({ organization, roles }) => (
									<div
										key={organization.id}
										className="grid grid-cols-[1fr_auto] items-center gap-2 text-sm"
									>
										<span className="text-gray-900">
											{organization.display_name || organization.name}
										</span>
										<div className="flex flex-wrap justify-end gap-1">
											{roles.length > 0 ? (
												roles.map((role) => (
													<Badge key={role} variant="secondary">
														{role}
													</Badge>
												))
											) : (
												<span className="text-gray-400">—</span>
											)}
										</div>
									</div>
								))}
							</div>
						) : (
							<p className="text-sm text-gray-400">No organizations</p>
						)}
					</div>
				)}
			</DialogContent>
		</Dialog>
	);
};

export default ViewUserModal;
