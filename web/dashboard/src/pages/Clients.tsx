import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import {
	ChevronsLeft,
	ChevronsRight,
	ChevronLeft,
	ChevronRight,
	ChevronDown,
	AlertCircle,
	Copy,
} from 'lucide-react';
import dayjs from 'dayjs';
import { toast } from 'sonner';
import UpdateClientModal from '../components/UpdateClientModal';
import DeleteClientModal from '../components/DeleteClientModal';
import RotateClientSecretModal from '../components/RotateClientSecretModal';
import { pageLimitsExtended, UpdateModalViews } from '../constants';
import { ClientsQuery } from '../graphql/queries';
import { copyTextToClipboard } from '../utils';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
import { Select } from '../components/ui/select';
import { Skeleton } from '../components/ui/skeleton';
import {
	DropdownMenu,
	DropdownMenuTrigger,
	DropdownMenuContent,
} from '../components/ui/dropdown-menu';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from '../components/ui/table';
import type { Client, ClientsResponse } from '../types';

interface PaginationProps {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

const Clients = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(false);
	const [clientData, setClientData] = useState<Client[]>([]);
	const [paginationProps, setPaginationProps] = useState<PaginationProps>({
		limit: 10,
		page: 1,
		offset: 0,
		total: 0,
		maxPages: 1,
	});

	const getMaxPages = (pagination: PaginationProps) => {
		const { limit, total } = pagination;
		if (total > 1) {
			return total % limit === 0
				? total / limit
				: Math.floor(total / limit) + 1;
		}
		return 1;
	};

	const fetchClientData = async () => {
		setLoading(true);
		const res = await client
			.query<ClientsResponse>(ClientsQuery, {
				params: {
					pagination: {
						pagination: {
							limit: paginationProps.limit,
							page: paginationProps.page,
						},
					},
				},
			})
			.toPromise();
		if (res.data?._clients) {
			const { pagination, clients } = res.data._clients;
			const maxPages = getMaxPages(pagination as unknown as PaginationProps);
			if (clients?.length) {
				setClientData(clients);
				setPaginationProps({
					...paginationProps,
					...pagination,
					maxPages,
				});
			} else {
				setClientData([]);
				if (paginationProps.page !== 1) {
					setPaginationProps({
						...paginationProps,
						...pagination,
						maxPages,
						page: 1,
					});
				}
			}
		}
		setLoading(false);
	};

	const paginationHandler = (value: Record<string, number>) => {
		setPaginationProps({ ...paginationProps, ...value });
	};

	useEffect(() => {
		fetchClientData();
	}, [paginationProps.page, paginationProps.limit]);

	const copyClientId = async (id: string) => {
		await copyTextToClipboard(id);
		toast.success('Client ID copied');
	};

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="flex items-center justify-between my-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">Clients</h1>
					<p className="mt-1 text-sm text-gray-500">
						Manage OAuth clients (machine and workload identities).
					</p>
				</div>
				<UpdateClientModal
					view={UpdateModalViews.ADD}
					fetchClients={fetchClientData}
				/>
			</div>
			{loading ? (
				<div className="min-h-[25vh] space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : clientData.length ? (
				<>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Client ID</TableHead>
								<TableHead>Name</TableHead>
								<TableHead>Allowed Scopes</TableHead>
								<TableHead>Active</TableHead>
								<TableHead>Created</TableHead>
								<TableHead>Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{clientData.map((clientItem) => (
								<TableRow key={clientItem.id}>
									<TableCell className="max-w-[220px] text-sm">
										<div className="flex items-center gap-2">
											<span className="truncate font-mono text-xs">
												{clientItem.id}
											</span>
											<button
												type="button"
												onClick={() => copyClientId(clientItem.id)}
												className="text-gray-400 hover:text-gray-600"
												aria-label="Copy client ID"
											>
												<Copy className="h-3 w-3" />
											</button>
										</div>
									</TableCell>
									<TableCell className="max-w-[220px] text-sm">
										{clientItem.name}
									</TableCell>
									<TableCell className="max-w-[300px]">
										<div className="flex flex-wrap gap-1">
											{clientItem.allowed_scopes.map((scope) => (
												<Badge key={scope} variant="secondary">
													{scope}
												</Badge>
											))}
										</div>
									</TableCell>
									<TableCell>
										<Badge
											variant={clientItem.is_active ? 'success' : 'warning'}
										>
											{clientItem.is_active.toString()}
										</Badge>
									</TableCell>
									<TableCell className="text-sm whitespace-nowrap">
										{clientItem.created_at
											? dayjs.unix(clientItem.created_at).format('MMM D, YYYY')
											: '—'}
									</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="sm">
													<span className="text-sm font-light">Menu</span>
													<ChevronDown className="ml-2 h-3 w-3" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<UpdateClientModal
													view={UpdateModalViews.Edit}
													selectedClient={clientItem}
													fetchClients={fetchClientData}
												/>
												<RotateClientSecretModal
													clientId={clientItem.id}
													clientName={clientItem.name}
												/>
												<DeleteClientModal
													clientId={clientItem.id}
													clientName={clientItem.name}
													fetchClients={fetchClientData}
												/>
											</DropdownMenuContent>
										</DropdownMenu>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>

					{/* Pagination */}
					{(paginationProps.maxPages > 1 || paginationProps.total >= 5) && (
						<div className="mt-4 flex items-center justify-between">
							<div className="flex gap-1">
								<Button
									variant="outline"
									size="icon"
									onClick={() => paginationHandler({ page: 1 })}
									disabled={paginationProps.page <= 1}
								>
									<ChevronsLeft className="h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									size="icon"
									onClick={() =>
										paginationHandler({
											page: paginationProps.page - 1,
										})
									}
									disabled={paginationProps.page <= 1}
								>
									<ChevronLeft className="h-4 w-4" />
								</Button>
							</div>

							<div className="flex items-center gap-4 text-sm">
								<span>
									Page <strong>{paginationProps.page}</strong> of{' '}
									<strong>{paginationProps.maxPages}</strong>
								</span>
								<div className="flex items-center gap-1">
									<span>Go to:</span>
									<Input
										type="number"
										min={1}
										max={paginationProps.maxPages}
										value={paginationProps.page}
										onChange={(e) =>
											paginationHandler({
												page: parseInt(e.target.value) || 1,
											})
										}
										className="h-8 w-16"
									/>
								</div>
								<Select
									value={paginationProps.limit}
									onChange={(e) =>
										paginationHandler({
											page: 1,
											limit: parseInt(e.target.value),
										})
									}
									className="h-8 w-28"
								>
									{pageLimitsExtended.map((pageSize) => (
										<option key={pageSize} value={pageSize}>
											Show {pageSize}
										</option>
									))}
								</Select>
							</div>

							<div className="flex gap-1">
								<Button
									variant="outline"
									size="icon"
									onClick={() =>
										paginationHandler({
											page: paginationProps.page + 1,
										})
									}
									disabled={paginationProps.page >= paginationProps.maxPages}
								>
									<ChevronRight className="h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									size="icon"
									onClick={() =>
										paginationHandler({
											page: paginationProps.maxPages,
										})
									}
									disabled={paginationProps.page >= paginationProps.maxPages}
								>
									<ChevronsRight className="h-4 w-4" />
								</Button>
							</div>
						</div>
					)}
				</>
			) : (
				<div className="flex min-h-[25vh] flex-col items-center justify-center text-gray-300">
					<AlertCircle className="h-16 w-16 mb-2" />
					<p className="text-2xl font-bold">No Data</p>
				</div>
			)}
		</div>
	);
};

export default Clients;
