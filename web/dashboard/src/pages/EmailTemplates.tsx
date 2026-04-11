import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import dayjs from 'dayjs';
import {
	ChevronsLeft,
	ChevronsRight,
	ChevronLeft,
	ChevronRight,
	ChevronDown,
	AlertCircle,
} from 'lucide-react';
import UpdateEmailTemplateModal from '../components/UpdateEmailTemplateModal';
import {
	pageLimitsExtended,
	UpdateModalViews,
	EmailTemplateInputDataFields,
} from '../constants';
import { EmailTemplatesQuery } from '../graphql/queries';
import DeleteEmailTemplateModal from '../components/DeleteEmailTemplateModal';
import { Button } from '../components/ui/button';
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
import type { EmailTemplate, EmailTemplatesResponse } from '../types';

interface PaginationProps {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

const EmailTemplates = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(false);
	const [emailTemplatesData, setEmailTemplatesData] = useState<EmailTemplate[]>(
		[],
	);
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

	const fetchEmailTemplatesData = async () => {
		setLoading(true);
		const res = await client
			.query<EmailTemplatesResponse>(EmailTemplatesQuery, {
				params: {
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (res.data?._email_templates) {
			const { pagination, email_templates: emailTemplates } =
				res.data._email_templates;
			const maxPages = getMaxPages(pagination as unknown as PaginationProps);
			if (emailTemplates?.length) {
				setEmailTemplatesData(emailTemplates);
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
		fetchEmailTemplatesData();
	}, [paginationProps.page, paginationProps.limit]);

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="flex items-center justify-between my-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">
						Email Templates
					</h1>
					<p className="mt-1 text-sm text-gray-500">
						Customize email templates for authentication events.
					</p>
				</div>
				<UpdateEmailTemplateModal
					view={UpdateModalViews.ADD}
					fetchEmailTemplatesData={fetchEmailTemplatesData}
				/>
			</div>
			{loading ? (
				<div className="min-h-[25vh] space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : emailTemplatesData.length ? (
				<>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Event Name</TableHead>
								<TableHead>Subject</TableHead>
								<TableHead>Created At</TableHead>
								<TableHead>Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{emailTemplatesData.map((templateData) => (
								<TableRow key={templateData.id}>
									<TableCell className="max-w-[300px] text-sm">
										{templateData.event_name}
									</TableCell>
									<TableCell className="text-sm">
										{templateData.subject}
									</TableCell>
									<TableCell className="text-sm">
										{dayjs(templateData.created_at * 1000).format(
											'MMM DD, YYYY',
										)}
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
												<UpdateEmailTemplateModal
													view={UpdateModalViews.Edit}
													selectedTemplate={templateData}
													fetchEmailTemplatesData={fetchEmailTemplatesData}
												/>
												<DeleteEmailTemplateModal
													emailTemplateId={templateData.id}
													eventName={templateData.event_name}
													fetchEmailTemplatesData={fetchEmailTemplatesData}
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

export default EmailTemplates;
