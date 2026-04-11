import React, { useEffect, useState } from 'react';
import dayjs from 'dayjs';
import { useClient } from 'urql';
import {
	ChevronsLeft,
	ChevronsRight,
	ChevronLeft,
	ChevronRight,
	Copy,
	AlertCircle,
} from 'lucide-react';
import { toast } from 'sonner';
import { copyTextToClipboard } from '../utils';
import { WebhookLogsQuery } from '../graphql/queries';
import { pageLimits } from '../constants';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Select } from './ui/select';
import { Input } from './ui/input';
import { Skeleton } from './ui/skeleton';
import { Tooltip, TooltipTrigger, TooltipContent } from './ui/tooltip';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from './ui/table';
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from './ui/dialog';

interface PaginationProps {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

interface WebhookLogData {
	id: string;
	http_status: number;
	request: string;
	response: string;
	created_at: number;
}

interface ViewWebhookLogsModalProps {
	webhookId: string;
	eventName: string;
}

const ViewWebhookLogsModal = ({
	webhookId,
	eventName,
}: ViewWebhookLogsModalProps) => {
	const client = useClient();
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState<boolean>(false);
	const [webhookLogs, setWebhookLogs] = useState<WebhookLogData[]>([]);
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

	const fetchWebhookLogsData = async () => {
		setLoading(true);
		const res = await client
			.query(WebhookLogsQuery, {
				params: {
					webhook_id: webhookId,
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (res.data?._webhook_logs) {
			const { pagination, webhook_logs } = res.data._webhook_logs;
			const maxPages = getMaxPages(pagination);
			if (webhook_logs?.length) {
				setWebhookLogs(webhook_logs);
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
		if (open) fetchWebhookLogsData();
	}, [open, paginationProps.page, paginationProps.limit]);

	return (
		<Dialog open={open} onOpenChange={setOpen}>
			<DialogTrigger asChild>
				<button className="w-full text-left px-2 py-1.5 text-sm hover:bg-gray-100 rounded-sm">
					View Logs
				</button>
			</DialogTrigger>
			<DialogContent className="max-w-4xl">
				<DialogHeader>
					<DialogTitle>Webhook Logs - {eventName}</DialogTitle>
				</DialogHeader>

				<div className="rounded-md border border-gray-200 p-4">
					{loading ? (
						<div className="space-y-3">
							{[1, 2, 3].map((i) => (
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : webhookLogs.length ? (
						<>
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>ID</TableHead>
										<TableHead>Created At</TableHead>
										<TableHead>Http Status</TableHead>
										<TableHead>Request</TableHead>
										<TableHead>Response</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{webhookLogs.map((logData) => (
										<TableRow key={logData.id}>
											<TableCell className="text-sm">
												{`${logData.id.substring(0, 5)}***${logData.id.substring(logData.id.length - 5)}`}
											</TableCell>
											<TableCell>
												{dayjs(logData.created_at * 1000).format(
													'MMM DD, YYYY',
												)}
											</TableCell>
											<TableCell>
												<Badge
													variant={
														logData.http_status >= 400
															? 'destructive'
															: 'success'
													}
												>
													{logData.http_status}
												</Badge>
											</TableCell>
											<TableCell>
												<div className="flex items-center gap-1">
													<Tooltip>
														<TooltipTrigger>
															<Badge
																variant={
																	logData.request ? 'secondary' : 'warning'
																}
															>
																{logData.request ? 'Payload' : 'No Data'}
															</Badge>
														</TooltipTrigger>
														<TooltipContent className="max-w-xs">
															<p className="break-all text-xs">
																{logData.request || 'null'}
															</p>
														</TooltipContent>
													</Tooltip>
													{logData.request && (
														<Button
															variant="ghost"
															size="icon"
															className="h-6 w-6"
															onClick={() => {
																copyTextToClipboard(logData.request);
																toast.success('Copied to clipboard');
															}}
														>
															<Copy className="h-3 w-3" />
														</Button>
													)}
												</div>
											</TableCell>
											<TableCell>
												<div className="flex items-center gap-1">
													<Tooltip>
														<TooltipTrigger>
															<Badge
																variant={
																	logData.response ? 'secondary' : 'warning'
																}
															>
																{logData.response ? 'Preview' : 'No Data'}
															</Badge>
														</TooltipTrigger>
														<TooltipContent className="max-w-xs">
															<p className="break-all text-xs">
																{logData.response || 'null'}
															</p>
														</TooltipContent>
													</Tooltip>
													{logData.response && (
														<Button
															variant="ghost"
															size="icon"
															className="h-6 w-6"
															onClick={() => {
																copyTextToClipboard(logData.response);
																toast.success('Copied to clipboard');
															}}
														>
															<Copy className="h-3 w-3" />
														</Button>
													)}
												</div>
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
											<span className="whitespace-nowrap">Go to:</span>
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
												paginationProps.page >= paginationProps.maxPages
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
												paginationProps.page >= paginationProps.maxPages
											}
										>
											<ChevronsRight className="h-4 w-4" />
										</Button>
									</div>
								</div>
							)}
						</>
					) : (
						<div className="flex min-h-[25vh] flex-col items-center justify-center text-gray-400">
							<AlertCircle className="h-16 w-16 mb-2" />
							<p className="text-xl font-bold">No Data</p>
						</div>
					)}
				</div>

				<DialogFooter>
					<Button onClick={() => setOpen(false)}>Close</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
};

export default ViewWebhookLogsModal;
