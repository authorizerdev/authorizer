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
import UpdateWebhookModal from '../components/UpdateWebhookModal';
import {
	pageLimits,
	WebhookInputDataFields,
	UpdateModalViews,
} from '../constants';
import { WebhooksDataQuery } from '../graphql/queries';
import DeleteWebhookModal from '../components/DeleteWebhookModal';
import ViewWebhookLogsModal from '../components/ViewWebhookLogsModal';
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
import type { Webhook, WebhooksResponse } from '../types';

interface PaginationProps {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

const Webhooks = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(false);
	const [webhookData, setWebhookData] = useState<Webhook[]>([]);
	const [paginationProps, setPaginationProps] = useState<PaginationProps>({
		limit: 5,
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

	const fetchWebookData = async () => {
		setLoading(true);
		const res = await client
			.query<WebhooksResponse>(WebhooksDataQuery, {
				params: {
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (res.data?._webhooks) {
			const { pagination, webhooks } = res.data._webhooks;
			const maxPages = getMaxPages(
				pagination as unknown as PaginationProps,
			);
			if (webhooks?.length) {
				setWebhookData(webhooks);
				setPaginationProps({
					...paginationProps,
					...pagination,
					maxPages,
				});
			} else {
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
		fetchWebookData();
	}, [paginationProps.page, paginationProps.limit]);

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="flex items-center justify-between my-4">
				<h2 className="text-base font-bold">Webhooks</h2>
				<UpdateWebhookModal
					view={UpdateModalViews.ADD}
					fetchWebookData={fetchWebookData}
				/>
			</div>
			{loading ? (
				<div className="min-h-[25vh] space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : webhookData.length ? (
				<>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Event Name</TableHead>
								<TableHead>Event Description</TableHead>
								<TableHead>Endpoint</TableHead>
								<TableHead>Enabled</TableHead>
								<TableHead>Headers</TableHead>
								<TableHead>Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{webhookData.map((webhook) => (
								<TableRow key={webhook.id}>
									<TableCell className="max-w-[300px] text-sm">
										{webhook.event_name.split('-')[0]}
									</TableCell>
									<TableCell className="max-w-[300px] text-sm">
										{webhook.event_description}
									</TableCell>
									<TableCell className="text-sm">
										{webhook.endpoint}
									</TableCell>
									<TableCell>
										<Badge
											variant={
												webhook.enabled
													? 'success'
													: 'warning'
											}
										>
											{webhook.enabled.toString()}
										</Badge>
									</TableCell>
									<TableCell>
										<Tooltip>
											<TooltipTrigger>
												<Badge variant="secondary">
													{Object.keys(
														webhook.headers || {},
													).length.toString()}
												</Badge>
											</TooltipTrigger>
											<TooltipContent>
												<pre className="text-xs">
													{JSON.stringify(
														webhook.headers,
														null,
														2,
													)}
												</pre>
											</TooltipContent>
										</Tooltip>
									</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button
													variant="ghost"
													size="sm"
												>
													<span className="text-sm font-light">
														Menu
													</span>
													<ChevronDown className="ml-2 h-3 w-3" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<UpdateWebhookModal
													view={
														UpdateModalViews.Edit
													}
													selectedWebhook={webhook}
													fetchWebookData={
														fetchWebookData
													}
												/>
												<DeleteWebhookModal
													webhookId={webhook.id}
													eventName={
														webhook.event_name
													}
													fetchWebookData={
														fetchWebookData
													}
												/>
												<ViewWebhookLogsModal
													webhookId={webhook.id}
													eventName={
														webhook.event_name
													}
												/>
											</DropdownMenuContent>
										</DropdownMenu>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>

					{/* Pagination */}
					{(paginationProps.maxPages > 1 ||
						paginationProps.total >= 5) && (
						<div className="mt-4 flex items-center justify-between">
							<div className="flex gap-1">
								<Button
									variant="outline"
									size="icon"
									onClick={() =>
										paginationHandler({ page: 1 })
									}
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
									Page{' '}
									<strong>{paginationProps.page}</strong> of{' '}
									<strong>
										{paginationProps.maxPages}
									</strong>
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
												page:
													parseInt(e.target.value) ||
													1,
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
									{pageLimits.map((pageSize) => (
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
									disabled={
										paginationProps.page >=
										paginationProps.maxPages
									}
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
									disabled={
										paginationProps.page >=
										paginationProps.maxPages
									}
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

export default Webhooks;
