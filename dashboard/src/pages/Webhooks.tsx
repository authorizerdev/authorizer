import React, { useEffect, useState } from 'react';
import { useClient } from 'urql';
import {
	Box,
	Button,
	Center,
	Flex,
	IconButton,
	Menu,
	MenuButton,
	MenuItem,
	MenuList,
	NumberDecrementStepper,
	NumberIncrementStepper,
	NumberInput,
	NumberInputField,
	NumberInputStepper,
	Select,
	Spinner,
	Table,
	TableCaption,
	Tag,
	Tbody,
	Td,
	Text,
	Th,
	Thead,
	Tooltip,
	Tr,
} from '@chakra-ui/react';
import {
	FaAngleDoubleLeft,
	FaAngleDoubleRight,
	FaAngleDown,
	FaAngleLeft,
	FaAngleRight,
	FaExclamationCircle,
} from 'react-icons/fa';
import UpdateWebhookModal from '../components/UpdateWebhookModal';
import {
	pageLimits,
	WebhookInputDataFields,
	UpdateWebhookModalViews,
} from '../constants';
import { WebhooksDataQuery } from '../graphql/queries';
import DeleteWebhookModal from '../components/DeleteWebhookModal';
import ViewWebhookLogsModal from '../components/ViewWebhookLogsModal';

interface paginationPropTypes {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

interface webhookDataTypes {
	[WebhookInputDataFields.ID]: string;
	[WebhookInputDataFields.EVENT_NAME]: string;
	[WebhookInputDataFields.ENDPOINT]: string;
	[WebhookInputDataFields.ENABLED]: boolean;
	[WebhookInputDataFields.HEADERS]?: Record<string, string>;
}

const Webhooks = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(false);
	const [webhookData, setWebhookData] = useState<webhookDataTypes[]>([]);
	const [paginationProps, setPaginationProps] = useState<paginationPropTypes>({
		limit: 5,
		page: 1,
		offset: 0,
		total: 0,
		maxPages: 1,
	});
	const getMaxPages = (pagination: paginationPropTypes) => {
		const { limit, total } = pagination;
		if (total > 1) {
			return total % limit === 0
				? total / limit
				: parseInt(`${total / limit}`) + 1;
		} else return 1;
	};
	const fetchWebookData = async () => {
		setLoading(true);
		const res = await client
			.query(WebhooksDataQuery, {
				params: {
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (res.data?._webhooks) {
			const { pagination, webhooks } = res.data?._webhooks;
			const maxPages = getMaxPages(pagination);
			if (webhooks?.length) {
				setWebhookData(webhooks);
				setPaginationProps({ ...paginationProps, ...pagination, maxPages });
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
	}, []);
	useEffect(() => {
		fetchWebookData();
	}, [paginationProps.page, paginationProps.limit]);
	return (
		<Box m="5" py="5" px="10" bg="white" rounded="md">
			<Flex margin="2% 0" justifyContent="space-between" alignItems="center">
				<Text fontSize="md" fontWeight="bold">
					Webhooks
				</Text>
				<UpdateWebhookModal
					view={UpdateWebhookModalViews.ADD}
					fetchWebookData={fetchWebookData}
				/>
			</Flex>
			{!loading ? (
				webhookData.length ? (
					<Table variant="simple">
						<Thead>
							<Tr>
								<Th>Event Name</Th>
								<Th>Endpoint</Th>
								<Th>Enabled</Th>
								<Th>Headers</Th>
								<Th>Actions</Th>
							</Tr>
						</Thead>
						<Tbody>
							{webhookData.map((webhook: webhookDataTypes) => (
								<Tr
									key={webhook[WebhookInputDataFields.ID]}
									style={{ fontSize: 14 }}
								>
									<Td maxW="300">
										{webhook[WebhookInputDataFields.EVENT_NAME]}
									</Td>
									<Td>{webhook[WebhookInputDataFields.ENDPOINT]}</Td>
									<Td>
										<Tag
											size="sm"
											variant="outline"
											colorScheme={
												webhook[WebhookInputDataFields.ENABLED]
													? 'green'
													: 'yellow'
											}
										>
											{webhook[WebhookInputDataFields.ENABLED].toString()}
										</Tag>
									</Td>
									<Td>
										<Tooltip
											bg="gray.300"
											color="black"
											label={JSON.stringify(
												webhook[WebhookInputDataFields.HEADERS],
												null,
												' '
											)}
										>
											<Tag size="sm" variant="outline" colorScheme="gray">
												{Object.keys(
													webhook[WebhookInputDataFields.HEADERS] || {}
												)?.length.toString()}
											</Tag>
										</Tooltip>
									</Td>
									<Td>
										<Menu>
											<MenuButton as={Button} variant="unstyled" size="sm">
												<Flex
													justifyContent="space-between"
													alignItems="center"
												>
													<Text fontSize="sm" fontWeight="light">
														Menu
													</Text>
													<FaAngleDown style={{ marginLeft: 10 }} />
												</Flex>
											</MenuButton>
											<MenuList>
												<UpdateWebhookModal
													view={UpdateWebhookModalViews.Edit}
													selectedWebhook={webhook}
													fetchWebookData={fetchWebookData}
												/>
												<DeleteWebhookModal
													webhookId={webhook[WebhookInputDataFields.ID]}
													eventName={webhook[WebhookInputDataFields.EVENT_NAME]}
													fetchWebookData={fetchWebookData}
												/>
												<ViewWebhookLogsModal
													webhookId={webhook[WebhookInputDataFields.ID]}
													eventName={webhook[WebhookInputDataFields.EVENT_NAME]}
												/>
											</MenuList>
										</Menu>
									</Td>
								</Tr>
							))}
						</Tbody>
						{(paginationProps.maxPages > 1 || paginationProps.total >= 5) && (
							<TableCaption>
								<Flex
									justifyContent="space-between"
									alignItems="center"
									m="2% 0"
								>
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
											{pageLimits.map((pageSize) => (
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
				)
			) : (
				<Center minH="25vh">
					<Spinner />
				</Center>
			)}
		</Box>
	);
};

export default Webhooks;
