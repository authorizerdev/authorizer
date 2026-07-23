import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import {
	ChevronsLeft,
	ChevronsRight,
	ChevronLeft,
	ChevronRight,
	ChevronDown,
	AlertCircle,
} from 'lucide-react';
import UpdateTrustedIssuerModal from '../components/UpdateTrustedIssuerModal';
import DeleteTrustedIssuerModal from '../components/DeleteTrustedIssuerModal';
import { pageLimitsExtended, UpdateModalViews } from '../constants';
import { TrustedIssuersQuery } from '../graphql/queries';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
import { Select } from '../components/ui/select';
import { Skeleton } from '../components/ui/skeleton';
import {
	Tooltip,
	TooltipTrigger,
	TooltipContent,
} from '../components/ui/tooltip';
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
import type { TrustedIssuer, TrustedIssuersResponse } from '../types';

interface PaginationProps {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

const TrustedIssuers = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(false);
	const [issuerData, setIssuerData] = useState<TrustedIssuer[]>([]);
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

	const fetchIssuerData = async () => {
		setLoading(true);
		const res = await client
			.query<TrustedIssuersResponse>(TrustedIssuersQuery, {
				params: {
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (res.data?._trusted_issuers) {
			const { pagination, trusted_issuers } = res.data._trusted_issuers;
			const maxPages = getMaxPages(pagination as unknown as PaginationProps);
			if (trusted_issuers?.length) {
				setIssuerData(trusted_issuers);
				setPaginationProps({
					...paginationProps,
					...pagination,
					maxPages,
				});
			} else {
				setIssuerData([]);
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
		fetchIssuerData();
	}, [paginationProps.page, paginationProps.limit]);

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="flex items-center justify-between my-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">
						Trusted Issuers
					</h1>
					<p className="mt-1 text-sm text-gray-500">
						External JWT issuers bound to service accounts for workload
						authentication.
					</p>
				</div>
				<UpdateTrustedIssuerModal
					view={UpdateModalViews.ADD}
					fetchIssuers={fetchIssuerData}
				/>
			</div>
			{loading ? (
				<div className="min-h-[25vh] space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : issuerData.length ? (
				<>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Issuer URL</TableHead>
								<TableHead>Type</TableHead>
								<TableHead>Expected Audience</TableHead>
								<TableHead>Allowed Subjects</TableHead>
								<TableHead>Active</TableHead>
								<TableHead>Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{issuerData.map((issuer) => (
								<TableRow key={issuer.id}>
									<TableCell className="max-w-[200px] text-sm">
										{issuer.name}
									</TableCell>
									<TableCell className="max-w-[260px] truncate text-sm">
										{issuer.issuer_url}
									</TableCell>
									<TableCell>
										<Badge variant="secondary">{issuer.issuer_type}</Badge>
									</TableCell>
									<TableCell className="max-w-[220px] truncate text-sm">
										{issuer.expected_aud}
									</TableCell>
									<TableCell>
										<Tooltip>
											<TooltipTrigger>
												<Badge variant="secondary">
													{issuer.allowed_subjects
														? issuer.allowed_subjects
																.split(',')
																.filter((s) => s.trim())
																.length.toString()
														: '0'}
												</Badge>
											</TooltipTrigger>
											<TooltipContent>
												<pre className="text-xs">
													{issuer.allowed_subjects
														? issuer.allowed_subjects
																.split(',')
																.map((s) => s.trim())
																.filter(Boolean)
																.join('\n')
														: 'No subjects allowed (deny-all)'}
												</pre>
											</TooltipContent>
										</Tooltip>
									</TableCell>
									<TableCell>
										<Badge variant={issuer.is_active ? 'success' : 'warning'}>
											{issuer.is_active.toString()}
										</Badge>
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
												<UpdateTrustedIssuerModal
													view={UpdateModalViews.Edit}
													selectedIssuer={issuer}
													fetchIssuers={fetchIssuerData}
												/>
												<DeleteTrustedIssuerModal
													issuerId={issuer.id}
													issuerName={issuer.name}
													fetchIssuers={fetchIssuerData}
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

export default TrustedIssuers;
