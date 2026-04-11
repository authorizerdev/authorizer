import React, { useCallback, useEffect, useState } from 'react';
import { useClient } from 'urql';
import {
	ScrollText,
	Filter,
	X,
	ChevronDown,
	ChevronRight,
	ChevronLeft,
	ChevronsLeft,
	ChevronsRight,
} from 'lucide-react';
import dayjs from 'dayjs';
import {
	Card,
	CardContent,
	CardHeader,
	CardTitle,
} from '../components/ui/card';
import { Badge } from '../components/ui/badge';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Skeleton } from '../components/ui/skeleton';
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from '../components/ui/table';
import { AuditLogsQuery } from '../graphql/queries';
import type { AuditLogsResponse, AuditLog, PaginationInfo } from '../types';
import { auditActionCategories, auditResourceTypes } from '../constants';
import { toast } from 'sonner';

const PAGE_SIZE = 20;

const AuditLogs = () => {
	const client = useClient();

	const [loading, setLoading] = useState(true);
	const [logs, setLogs] = useState<AuditLog[]>([]);
	const [pagination, setPagination] = useState<PaginationInfo>({
		limit: PAGE_SIZE,
		page: 1,
		offset: 0,
		total: 0,
	});
	const [expandedRow, setExpandedRow] = useState<string | null>(null);

	const [actionFilter, setActionFilter] = useState('');
	const [actorTypeFilter, setActorTypeFilter] = useState('');
	const [resourceTypeFilter, setResourceTypeFilter] = useState('');
	const [fromDate, setFromDate] = useState('');
	const [toDate, setToDate] = useState('');

	const fetchLogs = useCallback(
		async (page: number) => {
			setLoading(true);
			try {
				const params: Record<string, unknown> = {
					limit: PAGE_SIZE,
					page,
				};

				if (actionFilter) params.action = actionFilter;
				if (actorTypeFilter) params.actor_type = actorTypeFilter;
				if (resourceTypeFilter) params.resource_type = resourceTypeFilter;
				if (fromDate) params.from_timestamp = dayjs(fromDate).unix();
				if (toDate) params.to_timestamp = dayjs(toDate).endOf('day').unix();

				const res = await client
					.query<AuditLogsResponse>(AuditLogsQuery, { params })
					.toPromise();

				if (res.data?._audit_logs) {
					setLogs(res.data._audit_logs.audit_logs || []);
					setPagination(res.data._audit_logs.pagination);
				}

				if (res.error) {
					toast.error('Failed to load audit logs');
				}
			} catch {
				toast.error('Failed to load audit logs');
			} finally {
				setLoading(false);
			}
		},
		[
			client,
			actionFilter,
			actorTypeFilter,
			resourceTypeFilter,
			fromDate,
			toDate,
		],
	);

	useEffect(() => {
		fetchLogs(1);
	}, [fetchLogs]);

	const totalPages = Math.ceil(pagination.total / PAGE_SIZE);

	const handleClearFilters = () => {
		setActionFilter('');
		setActorTypeFilter('');
		setResourceTypeFilter('');
		setFromDate('');
		setToDate('');
	};

	const hasActiveFilters =
		actionFilter || actorTypeFilter || resourceTypeFilter || fromDate || toDate;

	const getActionBadgeVariant = (
		action: string,
	):
		| 'default'
		| 'success'
		| 'destructive'
		| 'warning'
		| 'secondary'
		| 'outline' => {
		if (
			action.includes('success') ||
			action.includes('signup') ||
			action.includes('created')
		)
			return 'success';
		if (
			action.includes('failed') ||
			action.includes('revoked') ||
			action.includes('deleted')
		)
			return 'destructive';
		if (action.startsWith('admin.')) return 'default';
		return 'secondary';
	};

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-semibold text-gray-900">Audit Logs</h1>
				<p className="mt-1 text-sm text-gray-500">
					Track all actions performed in your Authorizer instance.
				</p>
			</div>

			<Card>
				<CardHeader className="pb-3">
					<div className="flex items-center justify-between">
						<CardTitle className="flex items-center gap-2 text-sm font-medium text-gray-700">
							<Filter className="h-4 w-4" />
							Filters
						</CardTitle>
						{hasActiveFilters && (
							<Button
								variant="ghost"
								size="sm"
								onClick={handleClearFilters}
								className="text-gray-500 hover:text-gray-700"
							>
								<X className="mr-1 h-3 w-3" />
								Clear
							</Button>
						)}
					</div>
				</CardHeader>
				<CardContent>
					<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
						<div>
							<label className="mb-1 block text-xs font-medium text-gray-500">
								Action
							</label>
							<select
								value={actionFilter}
								onChange={(e) => setActionFilter(e.target.value)}
								className="flex h-9 w-full rounded-md border border-gray-200 bg-white px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-500"
							>
								<option value="">All actions</option>
								{Object.entries(auditActionCategories).map(
									([category, actions]) => (
										<optgroup key={category} label={category}>
											{actions.map((action) => (
												<option key={action} value={action}>
													{action}
												</option>
											))}
										</optgroup>
									),
								)}
							</select>
						</div>

						<div>
							<label className="mb-1 block text-xs font-medium text-gray-500">
								Actor Type
							</label>
							<select
								value={actorTypeFilter}
								onChange={(e) => setActorTypeFilter(e.target.value)}
								className="flex h-9 w-full rounded-md border border-gray-200 bg-white px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-500"
							>
								<option value="">All</option>
								<option value="user">User</option>
								<option value="admin">Admin</option>
							</select>
						</div>

						<div>
							<label className="mb-1 block text-xs font-medium text-gray-500">
								Resource Type
							</label>
							<select
								value={resourceTypeFilter}
								onChange={(e) => setResourceTypeFilter(e.target.value)}
								className="flex h-9 w-full rounded-md border border-gray-200 bg-white px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-500"
							>
								<option value="">All</option>
								{auditResourceTypes.map((type) => (
									<option key={type} value={type}>
										{type.replace('_', ' ')}
									</option>
								))}
							</select>
						</div>

						<div>
							<label className="mb-1 block text-xs font-medium text-gray-500">
								From
							</label>
							<Input
								type="date"
								value={fromDate}
								onChange={(e) => setFromDate(e.target.value)}
								className="h-9"
							/>
						</div>

						<div>
							<label className="mb-1 block text-xs font-medium text-gray-500">
								To
							</label>
							<Input
								type="date"
								value={toDate}
								onChange={(e) => setToDate(e.target.value)}
								className="h-9"
							/>
						</div>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardContent className="p-0">
					{loading ? (
						<div className="space-y-3 p-6">
							{Array.from({ length: 5 }).map((_, i) => (
								<Skeleton key={i} className="h-12 w-full" />
							))}
						</div>
					) : logs.length === 0 ? (
						<div className="flex flex-col items-center justify-center py-16 text-gray-500">
							<ScrollText className="mb-3 h-10 w-10 text-gray-300" />
							<p className="text-sm font-medium">No audit logs found</p>
							<p className="text-xs text-gray-400 mt-1">
								{hasActiveFilters
									? 'Try adjusting your filters.'
									: 'Activity will appear here as actions are performed.'}
							</p>
						</div>
					) : (
						<div className="overflow-x-auto">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead className="w-8"></TableHead>
										<TableHead>Timestamp</TableHead>
										<TableHead>Action</TableHead>
										<TableHead>Actor</TableHead>
										<TableHead>Resource</TableHead>
										<TableHead>IP Address</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{logs.map((log) => (
										<React.Fragment key={log.id}>
											<TableRow
												className="cursor-pointer hover:bg-gray-50"
												onClick={() =>
													setExpandedRow(expandedRow === log.id ? null : log.id)
												}
											>
												<TableCell className="w-8 pr-0">
													{expandedRow === log.id ? (
														<ChevronDown className="h-4 w-4 text-gray-400" />
													) : (
														<ChevronRight className="h-4 w-4 text-gray-400" />
													)}
												</TableCell>
												<TableCell className="whitespace-nowrap text-sm text-gray-600">
													{dayjs
														.unix(log.created_at)
														.format('MMM D, YYYY HH:mm:ss')}
												</TableCell>
												<TableCell>
													<Badge variant={getActionBadgeVariant(log.action)}>
														{log.action}
													</Badge>
												</TableCell>
												<TableCell>
													<div className="flex items-center gap-2">
														<span className="text-sm text-gray-900">
															{log.actor_email || '—'}
														</span>
														{log.actor_type && (
															<Badge variant="outline" className="text-xs">
																{log.actor_type}
															</Badge>
														)}
													</div>
												</TableCell>
												<TableCell>
													<div className="flex items-center gap-2">
														{log.resource_type && (
															<Badge variant="secondary" className="text-xs">
																{log.resource_type}
															</Badge>
														)}
														<span className="text-xs text-gray-500 font-mono truncate max-w-[120px]">
															{log.resource_id || '—'}
														</span>
													</div>
												</TableCell>
												<TableCell className="text-sm text-gray-500">
													{log.ip_address || '—'}
												</TableCell>
											</TableRow>
											{expandedRow === log.id && (
												<TableRow className="bg-gray-50">
													<TableCell colSpan={6} className="py-3 px-8">
														<div className="grid gap-2 text-sm sm:grid-cols-2">
															<div>
																<span className="font-medium text-gray-500">
																	User Agent:
																</span>
																<p className="mt-0.5 text-gray-700 text-xs break-all">
																	{log.user_agent || '—'}
																</p>
															</div>
															<div>
																<span className="font-medium text-gray-500">
																	Actor ID:
																</span>
																<p className="mt-0.5 text-gray-700 text-xs font-mono">
																	{log.actor_id || '—'}
																</p>
															</div>
															{log.metadata && log.metadata !== '{}' && (
																<div className="sm:col-span-2">
																	<span className="font-medium text-gray-500">
																		Metadata:
																	</span>
																	<pre className="mt-1 rounded bg-gray-100 p-2 text-xs text-gray-700 overflow-x-auto">
																		{(() => {
																			try {
																				return JSON.stringify(
																					JSON.parse(log.metadata),
																					null,
																					2,
																				);
																			} catch {
																				return log.metadata;
																			}
																		})()}
																	</pre>
																</div>
															)}
														</div>
													</TableCell>
												</TableRow>
											)}
										</React.Fragment>
									))}
								</TableBody>
							</Table>
						</div>
					)}

					{!loading && logs.length > 0 && (
						<div className="flex items-center justify-between border-t border-gray-200 px-4 py-3">
							<p className="text-sm text-gray-500">
								Showing {(pagination.page - 1) * PAGE_SIZE + 1}–
								{Math.min(pagination.page * PAGE_SIZE, pagination.total)} of{' '}
								{pagination.total}
							</p>
							<div className="flex items-center gap-1">
								<Button
									variant="outline"
									size="icon"
									className="h-8 w-8"
									disabled={pagination.page <= 1}
									onClick={() => fetchLogs(1)}
								>
									<ChevronsLeft className="h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									size="icon"
									className="h-8 w-8"
									disabled={pagination.page <= 1}
									onClick={() => fetchLogs(pagination.page - 1)}
								>
									<ChevronLeft className="h-4 w-4" />
								</Button>
								<span className="px-3 text-sm text-gray-700">
									Page {pagination.page} of {totalPages}
								</span>
								<Button
									variant="outline"
									size="icon"
									className="h-8 w-8"
									disabled={pagination.page >= totalPages}
									onClick={() => fetchLogs(pagination.page + 1)}
								>
									<ChevronRight className="h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									size="icon"
									className="h-8 w-8"
									disabled={pagination.page >= totalPages}
									onClick={() => fetchLogs(totalPages)}
								>
									<ChevronsRight className="h-4 w-4" />
								</Button>
							</div>
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
};

export default AuditLogs;
