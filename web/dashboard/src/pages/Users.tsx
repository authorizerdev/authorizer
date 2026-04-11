import React from 'react';
import { useClient } from 'urql';
import dayjs from 'dayjs';
import { toast } from 'sonner';
import {
	ChevronsLeft,
	ChevronsRight,
	ChevronLeft,
	ChevronRight,
	ChevronDown,
	AlertCircle,
	Search,
} from 'lucide-react';
import { UserDetailsQuery } from '../graphql/queries';
import { EnableAccess, RevokeAccess, UpdateUser } from '../graphql/mutation';
import { getGraphQLErrorMessage } from '../utils';
import EditUserModal from '../components/EditUserModal';
import DeleteUserModal from '../components/DeleteUserModal';
import InviteMembersModal from '../components/InviteMembersModal';
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
	DropdownMenuItem,
	DropdownMenuSeparator,
} from '../components/ui/dropdown-menu';
import {
	Table,
	TableHeader,
	TableBody,
	TableRow,
	TableHead,
	TableCell,
} from '../components/ui/table';
import type { User, UsersResponse } from '../types';

interface PaginationProps {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

const enum UpdateAccessActions {
	REVOKE = 'REVOKE',
	ENABLE = 'ENABLE',
}

const getMaxPages = (pagination: PaginationProps) => {
	const { limit, total } = pagination;
	if (total > 1) {
		return total % limit === 0
			? total / limit
			: Math.floor(total / limit) + 1;
	}
	return 1;
};

const PAGE_SIZE_OPTIONS = [10, 25, 50];

export default function Users() {
	const client = useClient();
	const [paginationProps, setPaginationProps] =
		React.useState<PaginationProps>({
			limit: 10,
			page: 1,
			offset: 0,
			total: 0,
			maxPages: 1,
		});
	const [userList, setUserList] = React.useState<User[]>([]);
	const [loading, setLoading] = React.useState<boolean>(false);
	const [searchQuery, setSearchQuery] = React.useState('');

	const updateUserList = async () => {
		setLoading(true);
		const { data } = await client
			.query<UsersResponse>(UserDetailsQuery, {
				params: {
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (data?._users) {
			const { pagination, users } = data._users;
			const maxPages = getMaxPages(pagination as unknown as PaginationProps);
			if (users && users.length > 0) {
				setPaginationProps({ ...paginationProps, ...pagination, maxPages });
				setUserList(users);
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

	React.useEffect(() => {
		updateUserList();
	}, []);

	React.useEffect(() => {
		updateUserList();
	}, [paginationProps.page, paginationProps.limit]);

	const paginationHandler = (value: Record<string, number>) => {
		setPaginationProps({ ...paginationProps, ...value });
	};

	const userVerificationHandler = async (user: User) => {
		const { id, email, phone_number } = user;
		let params: Record<string, unknown> = {};
		if (email) {
			params = { id, email, email_verified: true };
		}
		if (phone_number) {
			params = { id, phone_number, phone_number_verified: true };
		}
		const res = await client
			.mutation(UpdateUser, { params })
			.toPromise();
		if (res.error) {
			toast.error(
				getGraphQLErrorMessage(res.error, 'User verification failed'),
			);
		} else if (res.data?._update_user?.id) {
			toast.success('User verification successful');
		}
		updateUserList();
	};

	const updateAccessHandler = async (
		id: string,
		action: UpdateAccessActions,
	) => {
		switch (action) {
			case UpdateAccessActions.ENABLE: {
				const enableAccessRes = await client
					.mutation(EnableAccess, { param: { user_id: id } })
					.toPromise();
				if (enableAccessRes.error) {
					toast.error(
						getGraphQLErrorMessage(
							enableAccessRes.error,
							'User access enable failed',
						),
					);
				} else {
					toast.success('User access enabled successfully');
				}
				updateUserList();
				break;
			}
			case UpdateAccessActions.REVOKE: {
				const revokeAccessRes = await client
					.mutation(RevokeAccess, { param: { user_id: id } })
					.toPromise();
				if (revokeAccessRes.error) {
					toast.error(
						getGraphQLErrorMessage(
							revokeAccessRes.error,
							'User access revoke failed',
						),
					);
				} else {
					toast.success('User access revoked successfully');
				}
				updateUserList();
				break;
			}
			default:
				break;
		}
	};

	const multiFactorAuthUpdateHandler = async (user: User) => {
		const res = await client
			.mutation(UpdateUser, {
				params: {
					id: user.id,
					is_multi_factor_auth_enabled:
						!user.is_multi_factor_auth_enabled,
				},
			})
			.toPromise();
		if (res.data?._update_user?.id) {
			toast.success(
				`Multi factor authentication ${
					user.is_multi_factor_auth_enabled ? 'disabled' : 'enabled'
				} for user`,
			);
			updateUserList();
			return;
		}
		if (res.error) {
			toast.error(
				getGraphQLErrorMessage(
					res.error,
					'Multi factor authentication update failed for user',
				),
			);
		}
	};

	const filteredUsers = userList.filter(
		(user) =>
			searchQuery === '' ||
			(user.email || '').toLowerCase().includes(searchQuery.toLowerCase()),
	);

	return (
		<div className="m-5 rounded-md bg-white py-5 px-10">
			<div className="flex items-center justify-between my-4">
				<div>
					<h1 className="text-2xl font-semibold text-gray-900">Users</h1>
					<p className="mt-1 text-sm text-gray-500">
						Manage users, roles, and access.
					</p>
				</div>
				<InviteMembersModal updateUserList={updateUserList} />
			</div>
			<div className="flex items-center gap-2 mb-4">
				<div className="relative flex-1 max-w-sm">
					<Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
					<Input
						placeholder="Search by email..."
						value={searchQuery}
						onChange={(e) => setSearchQuery(e.target.value)}
						className="pl-9"
					/>
				</div>
			</div>
			{loading ? (
				<div className="min-h-[25vh] space-y-3">
					{[1, 2, 3, 4, 5].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : filteredUsers.length > 0 ? (
				<>
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Email / Phone</TableHead>
								<TableHead>Created At</TableHead>
								<TableHead>Signup Methods</TableHead>
								<TableHead>Roles</TableHead>
								<TableHead>Verified</TableHead>
								<TableHead>Access</TableHead>
								<TableHead>
									<Tooltip>
										<TooltipTrigger>MFA</TooltipTrigger>
										<TooltipContent>
											MultiFactor Authentication
											Enabled/Disabled
										</TooltipContent>
									</Tooltip>
								</TableHead>
								<TableHead>Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{filteredUsers.map((user) => {
								const {
									email_verified,
									phone_number_verified,
									created_at,
									...rest
								} = user;
								return (
									<TableRow key={user.id}>
										<TableCell className="max-w-[300px] truncate text-sm">
											{user.email || user.phone_number}
										</TableCell>
										<TableCell className="text-sm">
											{dayjs(
												user.created_at * 1000,
											).format('MMM DD, YYYY')}
										</TableCell>
										<TableCell className="text-sm">
											{user.signup_methods}
										</TableCell>
										<TableCell className="text-sm">
											{user.roles.join(', ')}
										</TableCell>
										<TableCell>
											<Badge
												variant={
													user.email_verified ||
													user.phone_number_verified
														? 'success'
														: 'warning'
												}
											>
												{(
													user.email_verified ||
													user.phone_number_verified
												)?.toString()}
											</Badge>
										</TableCell>
										<TableCell>
											<Badge
												variant={
													user.revoked_timestamp
														? 'destructive'
														: 'success'
												}
											>
												{user.revoked_timestamp
													? 'Revoked'
													: 'Enabled'}
											</Badge>
										</TableCell>
										<TableCell>
											<Badge
												variant={
													user.is_multi_factor_auth_enabled
														? 'success'
														: 'destructive'
												}
											>
												{user.is_multi_factor_auth_enabled
													? 'Enabled'
													: 'Disabled'}
											</Badge>
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
													{!user.email_verified &&
														!user.phone_number_verified && (
															<DropdownMenuItem
																onClick={() =>
																	userVerificationHandler(
																		user,
																	)
																}
															>
																Verify User
															</DropdownMenuItem>
														)}
													<EditUserModal
														user={
															rest as unknown as {
																id: string;
																email: string;
																given_name: string;
																family_name: string;
																middle_name: string;
																nickname: string;
																gender: string;
																birthdate: string;
																phone_number: string;
																picture: string;
																roles: string[];
															}
														}
														updateUserList={
															updateUserList
														}
													/>
													<DeleteUserModal
														user={rest}
														updateUserList={
															updateUserList
														}
													/>
													<DropdownMenuSeparator />
													{user.revoked_timestamp ? (
														<DropdownMenuItem
															onClick={() =>
																updateAccessHandler(
																	user.id,
																	UpdateAccessActions.ENABLE,
																)
															}
														>
															Enable Access
														</DropdownMenuItem>
													) : (
														<DropdownMenuItem
															onClick={() =>
																updateAccessHandler(
																	user.id,
																	UpdateAccessActions.REVOKE,
																)
															}
														>
															Revoke Access
														</DropdownMenuItem>
													)}
													{user.is_multi_factor_auth_enabled ? (
														<DropdownMenuItem
															onClick={() =>
																multiFactorAuthUpdateHandler(
																	user,
																)
															}
														>
															Disable MFA
														</DropdownMenuItem>
													) : (
														<DropdownMenuItem
															onClick={() =>
																multiFactorAuthUpdateHandler(
																	user,
																)
															}
														>
															Enable MFA
														</DropdownMenuItem>
													)}
												</DropdownMenuContent>
											</DropdownMenu>
										</TableCell>
									</TableRow>
								);
							})}
						</TableBody>
					</Table>

					{/* Pagination */}
					{(paginationProps.maxPages > 1 ||
						paginationProps.total >= 10) && (
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
									<span className="whitespace-nowrap">
										Go to:
									</span>
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
									{PAGE_SIZE_OPTIONS.map((pageSize) => (
										<option
											key={pageSize}
											value={pageSize}
										>
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
}
