import React from 'react';
import { useClient } from 'urql';
import dayjs from 'dayjs';
import {
	Box,
	Flex,
	IconButton,
	NumberDecrementStepper,
	NumberIncrementStepper,
	NumberInput,
	NumberInputField,
	NumberInputStepper,
	Select,
	Table,
	Tag,
	Tbody,
	Td,
	Text,
	TableCaption,
	Th,
	Thead,
	Tooltip,
	Tr,
	Button,
	Center,
	Menu,
	MenuButton,
	MenuList,
	MenuItem,
} from '@chakra-ui/react';
import {
	FaAngleLeft,
	FaAngleRight,
	FaAngleDoubleLeft,
	FaAngleDoubleRight,
	FaExclamationCircle,
	FaAngleDown,
} from 'react-icons/fa';
import { UserDetailsQuery } from '../graphql/queries';
import { UpdateUser } from '../graphql/mutation';

interface paginationPropTypes {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

interface userDataTypes {
	id: string;
	email: string;
	email_verified: boolean;
	given_name: string;
	family_name: string;
	middle_name: string;
	nickname: string;
	gender: string;
	birthdate: string;
	phone_number: string;
	picture: string;
	roles: [string];
	created_at: number;
}

const getMaxPages = (pagination: paginationPropTypes) => {
	const { limit, total } = pagination;
	if (total > 1) {
		return total % limit === 0
			? total / limit
			: parseInt(`${total / limit}`) + 1;
	} else return 1;
};

const getLimits = (pagination: paginationPropTypes) => {
	const { total } = pagination;
	const limits = [5];
	if (total > 10) {
		for (let i = 10; i <= total && limits.length <= 10; i += 5) {
			limits.push(i);
		}
	}
	return limits;
};

export default function Users() {
	const client = useClient();
	const [paginationProps, setPaginationProps] =
		React.useState<paginationPropTypes>({
			limit: 5,
			page: 1,
			offset: 0,
			total: 0,
			maxPages: 1,
		});
	const [userList, setUserList] = React.useState<userDataTypes[]>([]);
	const updateData = async () => {
		const { data } = await client
			.query(UserDetailsQuery, {
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
			const maxPages = getMaxPages(pagination);
			setPaginationProps({ ...paginationProps, ...pagination, maxPages });
			setUserList(users);
			console.log('users ==>> ', users);
			console.log('pagination ==>> ', { ...pagination, maxPages });
		}
	};
	React.useEffect(() => {
		updateData();
	}, []);
	React.useEffect(() => {
		updateData();
	}, [paginationProps.page, paginationProps.limit]);

	const paginationHandler = (value: Record<string, number>) => {
		setPaginationProps({ ...paginationProps, ...value });
	};

	const userVerificationHandler = async (user: userDataTypes) => {
		const { id, email } = user;
		await client
			.mutation(UpdateUser, {
				params: {
					id,
					email,
					email_verified: true,
				},
			})
			.toPromise();
		updateData();
	};

	return (
		<Box m="5" py="5" px="10" bg="white" rounded="md">
			<Flex margin="2% 0" justifyContent="space-between" alignItems="center">
				<Text fontSize="md" fontWeight="bold">
					Users
				</Text>
			</Flex>
			{userList.length > 0 ? (
				<Table variant="simple">
					<Thead>
						<Tr>
							<Th>Email</Th>
							<Th>Created At</Th>
							<Th>Verified</Th>
							<Th>Actions</Th>
						</Tr>
					</Thead>
					<Tbody>
						{userList.map((user: userDataTypes) => (
							<Tr key={user.id} style={{ fontSize: 14 }}>
								<Td>{user.email}</Td>
								<Td>{dayjs(user.created_at).format('MMM DD, YYYY')}</Td>
								<Td>
									<Tag
										size="sm"
										variant="outline"
										colorScheme={user.email_verified ? 'green' : 'yellow'}
									>
										{user.email_verified.toString()}
									</Tag>
								</Td>
								<Td>
									<Menu>
										<MenuButton as={Button} variant="unstyled" size="sm">
											<Flex justifyContent="space-between" alignItems="center">
												<Text fontSize="sm" fontWeight="light">
													Menu
												</Text>
												<FaAngleDown style={{ marginLeft: 10 }} />
											</Flex>
										</MenuButton>
										<MenuList>
											<MenuItem value="update">Update User Details</MenuItem>
											{!user.email_verified && (
												<MenuItem
													value="verify"
													onClick={() => userVerificationHandler(user)}
												>
													Verify User
												</MenuItem>
											)}
										</MenuList>
									</Menu>
								</Td>
							</Tr>
						))}
					</Tbody>
					{paginationProps.maxPages > 1 && (
						<TableCaption>
							<Flex justifyContent="space-between" alignItems="center" m="2% 0">
								<Flex flex="1">
									<Tooltip label="First Page">
										<IconButton
											aria-label="icon button"
											onClick={() =>
												paginationHandler({
													page: 1,
												})
											}
											isDisabled={paginationProps.page <= 1}
											mr={4}
											icon={<FaAngleDoubleLeft />}
										/>
									</Tooltip>
									<Tooltip label="Previous Page">
										<IconButton
											aria-label="icon button"
											onClick={() =>
												paginationHandler({
													page: paginationProps.page - 1,
												})
											}
											isDisabled={paginationProps.page <= 1}
											icon={<FaAngleLeft />}
										/>
									</Tooltip>
								</Flex>
								<Flex
									flex="8"
									justifyContent="space-evenly"
									alignItems="center"
								>
									<Text mr={8}>
										Page{' '}
										<Text fontWeight="bold" as="span">
											{paginationProps.page}
										</Text>{' '}
										of{' '}
										<Text fontWeight="bold" as="span">
											{paginationProps.maxPages}
										</Text>
									</Text>
									<Flex alignItems="center">
										<Text flexShrink="0">Go to page:</Text>{' '}
										<NumberInput
											ml={2}
											mr={8}
											w={28}
											min={1}
											max={paginationProps.maxPages}
											onChange={(value) =>
												paginationHandler({
													page: parseInt(value),
												})
											}
											value={paginationProps.page}
										>
											<NumberInputField />
											<NumberInputStepper>
												<NumberIncrementStepper />
												<NumberDecrementStepper />
											</NumberInputStepper>
										</NumberInput>
									</Flex>
									<Select
										w={32}
										value={paginationProps.limit}
										onChange={(e) =>
											paginationHandler({
												page: 1,
												limit: parseInt(e.target.value),
											})
										}
									>
										{getLimits(paginationProps).map((pageSize) => (
											<option key={pageSize} value={pageSize}>
												Show {pageSize}
											</option>
										))}
									</Select>
								</Flex>
								<Flex flex="1">
									<Tooltip label="Next Page">
										<IconButton
											aria-label="icon button"
											onClick={() =>
												paginationHandler({
													page: paginationProps.page + 1,
												})
											}
											isDisabled={
												paginationProps.page >= paginationProps.maxPages
											}
											icon={<FaAngleRight />}
										/>
									</Tooltip>
									<Tooltip label="Last Page">
										<IconButton
											aria-label="icon button"
											onClick={() =>
												paginationHandler({
													page: paginationProps.maxPages,
												})
											}
											isDisabled={
												paginationProps.page >= paginationProps.maxPages
											}
											ml={4}
											icon={<FaAngleDoubleRight />}
										/>
									</Tooltip>
								</Flex>
							</Flex>
						</TableCaption>
					)}
				</Table>
			) : (
				<Flex
					flexDirection="column"
					minH="25vh"
					justifyContent="center"
					alignItems="center"
				>
					<Center w="50px" marginRight="1.5%">
						<FaExclamationCircle style={{ color: '#f0f0f0', fontSize: 70 }} />
					</Center>
					<Text
						fontSize="2xl"
						paddingRight="1%"
						fontWeight="bold"
						color="#d9d9d9"
					>
						No Data
					</Text>
				</Flex>
			)}
		</Box>
	);
}
