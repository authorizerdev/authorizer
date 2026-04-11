import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useClient } from 'urql';
import { Users, Activity, Copy, Check, ArrowRight } from 'lucide-react';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import {
	Card,
	CardContent,
	CardHeader,
	CardTitle,
} from '../components/ui/card';
import { Button } from '../components/ui/button';
import { Skeleton } from '../components/ui/skeleton';
import {
	UserDetailsQuery,
	MetaQuery,
	AuditLogsQuery,
} from '../graphql/queries';
import type {
	UsersResponse,
	MetaResponse,
	AuditLogsResponse,
	AuditLog,
} from '../types';
import { copyTextToClipboard } from '../utils';
import { toast } from 'sonner';

dayjs.extend(relativeTime);

const Overview = () => {
	const client = useClient();
	const navigate = useNavigate();

	const [loading, setLoading] = useState(true);
	const [totalUsers, setTotalUsers] = useState(0);
	const [meta, setMeta] = useState<MetaResponse['meta'] | null>(null);
	const [recentActivity, setRecentActivity] = useState<AuditLog[]>([]);
	const [copiedClientId, setCopiedClientId] = useState(false);

	useEffect(() => {
		const fetchData = async () => {
			setLoading(true);
			try {
				const [usersRes, metaRes, auditRes] = await Promise.all([
					client
						.query<UsersResponse>(UserDetailsQuery, {
							params: { pagination: { limit: 1, page: 1 } },
						})
						.toPromise(),
					client.query<MetaResponse>(MetaQuery, {}).toPromise(),
					client
						.query<AuditLogsResponse>(AuditLogsQuery, {
							params: { pagination: { limit: 5, page: 1 } },
						})
						.toPromise(),
				]);

				if (usersRes.data?._users) {
					setTotalUsers(usersRes.data._users.pagination.total);
				}

				if (metaRes.data?.meta) {
					setMeta(metaRes.data.meta);
				}

				if (auditRes.data?._audit_logs) {
					setRecentActivity(auditRes.data._audit_logs.audit_logs || []);
				}
			} catch {
				toast.error('Failed to load dashboard data');
			} finally {
				setLoading(false);
			}
		};

		fetchData();
	}, [client]);

	const handleCopyClientId = async () => {
		if (meta?.client_id) {
			await copyTextToClipboard(meta.client_id);
			setCopiedClientId(true);
			toast.success('Client ID copied');
			setTimeout(() => setCopiedClientId(false), 2000);
		}
	};

	const getActionColor = (action: string): string => {
		if (action.startsWith('admin.')) return 'bg-purple-100 text-purple-700';
		if (action.includes('login_success') || action.includes('signup'))
			return 'bg-green-100 text-green-700';
		if (action.includes('failed') || action.includes('revoked'))
			return 'bg-red-100 text-red-700';
		return 'bg-gray-100 text-gray-700';
	};

	if (loading) {
		return (
			<div className="m-5 rounded-md bg-white py-5 px-10 space-y-6">
				<div>
					<Skeleton className="h-8 w-48" />
					<Skeleton className="mt-1 h-4 w-72" />
				</div>
				<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					<Skeleton className="h-32" />
					<Skeleton className="h-32" />
					<Skeleton className="h-32" />
				</div>
				<Skeleton className="h-64" />
			</div>
		);
	}

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10 space-y-6">
			<div>
				<h1 className="text-2xl font-semibold text-gray-900">Overview</h1>
				<p className="mt-1 text-sm text-gray-500">
					Welcome to your Authorizer dashboard.
				</p>
			</div>

			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				<Card
					className="cursor-pointer hover:border-blue-200 transition-colors"
					onClick={() => navigate('/users')}
				>
					<CardHeader className="flex flex-row items-center justify-between pb-2">
						<CardTitle className="text-sm font-medium text-gray-500">
							Total Users
						</CardTitle>
						<Users className="h-4 w-4 text-gray-400" />
					</CardHeader>
					<CardContent>
						<div className="text-3xl font-bold text-gray-900">{totalUsers}</div>
						<p className="mt-1 text-xs text-blue-600">View all users →</p>
					</CardContent>
				</Card>

				<Card>
					<CardHeader className="flex flex-row items-center justify-between pb-2">
						<CardTitle className="text-sm font-medium text-gray-500">
							Version
						</CardTitle>
						<Activity className="h-4 w-4 text-gray-400" />
					</CardHeader>
					<CardContent>
						<div className="text-3xl font-bold text-gray-900">
							{meta?.version || '—'}
						</div>
						<p className="mt-1 text-xs text-gray-500">Authorizer server</p>
					</CardContent>
				</Card>

				<Card>
					<CardHeader className="flex flex-row items-center justify-between pb-2">
						<CardTitle className="text-sm font-medium text-gray-500">
							Client ID
						</CardTitle>
						<button
							onClick={handleCopyClientId}
							className="text-gray-400 hover:text-gray-600 transition-colors"
							aria-label="Copy client ID"
						>
							{copiedClientId ? (
								<Check className="h-4 w-4 text-green-500" />
							) : (
								<Copy className="h-4 w-4" />
							)}
						</button>
					</CardHeader>
					<CardContent>
						<div className="text-sm font-mono text-gray-900 truncate">
							{meta?.client_id || '—'}
						</div>
						<p className="mt-1 text-xs text-gray-500">Click icon to copy</p>
					</CardContent>
				</Card>
			</div>

			<Card>
				<CardHeader className="flex flex-row items-center justify-between">
					<CardTitle className="text-base font-semibold text-gray-900">
						Recent Activity
					</CardTitle>
					<Button
						variant="ghost"
						size="sm"
						className="text-blue-600 hover:text-blue-700"
						onClick={() => navigate('/audit-logs')}
					>
						View all
						<ArrowRight className="ml-1 h-3 w-3" />
					</Button>
				</CardHeader>
				<CardContent>
					{recentActivity.length === 0 ? (
						<p className="text-sm text-gray-500 py-4 text-center">
							No recent activity recorded.
						</p>
					) : (
						<div className="space-y-3">
							{recentActivity.map((log) => (
								<div
									key={log.id}
									className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
								>
									<div className="flex items-center gap-3 min-w-0">
										<span
											className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${getActionColor(log.action)}`}
										>
											{log.action}
										</span>
										<span className="text-sm text-gray-600 truncate">
											{log.actor_email || log.actor_id || '—'}
										</span>
									</div>
									<span className="text-xs text-gray-400 whitespace-nowrap ml-4">
										{dayjs.unix(log.created_at).fromNow()}
									</span>
								</div>
							))}
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
};

export default Overview;
