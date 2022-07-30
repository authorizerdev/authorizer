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
import UpdateEmailTemplateModal from '../components/UpdateEmailTemplateModal';
import {
	pageLimits,
	UpdateModalViews,
	EmailTemplateInputDataFields,
} from '../constants';
import { EmailTemplatesQuery, WebhooksDataQuery } from '../graphql/queries';
import dayjs from 'dayjs';

interface paginationPropTypes {
	limit: number;
	page: number;
	offset: number;
	total: number;
	maxPages: number;
}

interface EmailTemplateDataType {
	[EmailTemplateInputDataFields.ID]: string;
	[EmailTemplateInputDataFields.EVENT_NAME]: string;
	[EmailTemplateInputDataFields.SUBJECT]: string;
	[EmailTemplateInputDataFields.CREATED_AT]: number;
	[EmailTemplateInputDataFields.TEMPLATE]: string;
}

const EmailTemplates = () => {
	const client = useClient();
	const [loading, setLoading] = useState<boolean>(false);
	const [emailTemplatesData, setEmailTemplatesData] = useState<
		EmailTemplateDataType[]
	>([]);
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
	const fetchEmailTemplatesData = async () => {
		setLoading(true);
		const res = await client
			.query(EmailTemplatesQuery, {
				params: {
					pagination: {
						limit: paginationProps.limit,
						page: paginationProps.page,
					},
				},
			})
			.toPromise();
		if (res.data?._email_templates) {
			const { pagination, EmailTemplates: emailTemplates } =
				res.data?._email_templates;
			const maxPages = getMaxPages(pagination);
			if (emailTemplates?.length) {
				setEmailTemplatesData(emailTemplates);
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
		fetchEmailTemplatesData();
	}, [paginationProps.page, paginationProps.limit]);
	return (
		<Box m="5" py="5" px="10" bg="white" rounded="md">
			<Flex margin="2% 0" justifyContent="space-between" alignItems="center">
				<Text fontSize="md" fontWeight="bold">
					Email Templates
				</Text>
				<UpdateEmailTemplateModal
					view={UpdateModalViews.ADD}
					fetchEmailTemplatesData={fetchEmailTemplatesData}
				/>
			</Flex>
			{!loading ? (
				emailTemplatesData.length ? (
					<Table variant="simple">
						<Thead>
							<Tr>
								<Th>Event Name</Th>
								<Th>Subject</Th>
								<Th>Created At</Th>
								<Th>Actions</Th>
							</Tr>
						</Thead>
						<Tbody>
							{emailTemplatesData.map((templateData: EmailTemplateDataType) => (
								<Tr
									key={templateData[EmailTemplateInputDataFields.ID]}
									style={{ fontSize: 14 }}
								>
									<Td maxW="300">
										{templateData[EmailTemplateInputDataFields.EVENT_NAME]}
									</Td>
									<Td>{templateData[EmailTemplateInputDataFields.SUBJECT]}</Td>
									<Td>
										{dayjs(templateData.created_at * 1000).format(
											'MMM DD, YYYY'
										)}
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
												<UpdateEmailTemplateModal
													view={UpdateModalViews.Edit}
													selectedTemplate={templateData}
													fetchEmailTemplatesData={fetchEmailTemplatesData}
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

export default EmailTemplates;
