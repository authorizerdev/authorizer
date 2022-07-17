import React, { useEffect, useState } from 'react';
import {
	Button,
	Center,
	Flex,
	Input,
	InputGroup,
	InputRightElement,
	MenuItem,
	Modal,
	ModalBody,
	ModalCloseButton,
	ModalContent,
	ModalFooter,
	ModalHeader,
	ModalOverlay,
	Switch,
	Text,
	useDisclosure,
	useToast,
} from '@chakra-ui/react';
import { FaMinusCircle, FaPlus } from 'react-icons/fa';
import { useClient } from 'urql';
import {
	ArrayInputOperations,
	WebhookInputDataFields,
	WebhookInputHeaderFields,
	UpdateWebhookModalViews,
} from '../constants';
import {
	capitalizeFirstLetter,
	validateEventName,
	validateURI,
} from '../utils';
import { AddWebhook, EditWebhook } from '../graphql/mutation';

interface headersDataType {
	[WebhookInputHeaderFields.KEY]: string;
	[WebhookInputHeaderFields.VALUE]: string;
}

interface headersValidatorDataType {
	[WebhookInputHeaderFields.KEY]: boolean;
	[WebhookInputHeaderFields.VALUE]: boolean;
}

interface selecetdWebhookDataTypes {
	[WebhookInputDataFields.ID]: string;
	[WebhookInputDataFields.EVENT_NAME]: string;
	[WebhookInputDataFields.ENDPOINT]: string;
	[WebhookInputDataFields.ENABLED]: boolean;
	[WebhookInputDataFields.HEADERS]?: Record<string, string>;
}

interface UpdateWebhookModalInputPropTypes {
	view: UpdateWebhookModalViews;
	selectedWebhook?: selecetdWebhookDataTypes;
	fetchWebookData: Function;
}

const initHeadersData: headersDataType = {
	[WebhookInputHeaderFields.KEY]: '',
	[WebhookInputHeaderFields.VALUE]: '',
};

const initHeadersValidatorData: headersValidatorDataType = {
	[WebhookInputHeaderFields.KEY]: true,
	[WebhookInputHeaderFields.VALUE]: true,
};

interface webhookDataType {
	[WebhookInputDataFields.EVENT_NAME]: string;
	[WebhookInputDataFields.ENDPOINT]: string;
	[WebhookInputDataFields.ENABLED]: boolean;
	[WebhookInputDataFields.HEADERS]: headersDataType[];
}

interface validatorDataType {
	[WebhookInputDataFields.EVENT_NAME]: boolean;
	[WebhookInputDataFields.ENDPOINT]: boolean;
	[WebhookInputDataFields.HEADERS]: headersValidatorDataType[];
}

const initWebhookData: webhookDataType = {
	[WebhookInputDataFields.EVENT_NAME]: '',
	[WebhookInputDataFields.ENDPOINT]: '',
	[WebhookInputDataFields.ENABLED]: false,
	[WebhookInputDataFields.HEADERS]: [{ ...initHeadersData }],
};

const initWebhookValidatorData: validatorDataType = {
	[WebhookInputDataFields.EVENT_NAME]: true,
	[WebhookInputDataFields.ENDPOINT]: true,
	[WebhookInputDataFields.HEADERS]: [{ ...initHeadersValidatorData }],
};

const UpdateWebhookModal = ({
	view,
	selectedWebhook,
	fetchWebookData,
}: UpdateWebhookModalInputPropTypes) => {
	const client = useClient();
	const toast = useToast();
	const { isOpen, onOpen, onClose } = useDisclosure();
	const [loading, setLoading] = useState<boolean>(false);
	const [webhook, setWebhook] = useState<webhookDataType>({
		...initWebhookData,
	});
	const [validator, setValidator] = useState<validatorDataType>({
		...initWebhookValidatorData,
	});
	const inputChangehandler = (
		inputType: string,
		value: any,
		headerInputType: string = WebhookInputHeaderFields.KEY,
		headerIndex: number = 0
	) => {
		switch (inputType) {
			case WebhookInputDataFields.EVENT_NAME:
				setWebhook({ ...webhook, [inputType]: value });
				setValidator({
					...validator,
					[WebhookInputDataFields.EVENT_NAME]: validateEventName(value),
				});
				break;
			case WebhookInputDataFields.ENDPOINT:
				setWebhook({ ...webhook, [inputType]: value });
				setValidator({
					...validator,
					[WebhookInputDataFields.ENDPOINT]: validateURI(value),
				});
				break;
			case WebhookInputDataFields.ENABLED:
				setWebhook({ ...webhook, [inputType]: value });
				break;
			case WebhookInputDataFields.HEADERS:
				const updatedHeaders: any = [
					...webhook[WebhookInputDataFields.HEADERS],
				];
				const updatedHeadersValidatorData: any = [
					...validator[WebhookInputDataFields.HEADERS],
				];
				const otherHeaderInputType =
					headerInputType === WebhookInputHeaderFields.KEY
						? WebhookInputHeaderFields.VALUE
						: WebhookInputHeaderFields.KEY;
				updatedHeaders[headerIndex][headerInputType] = value;
				updatedHeadersValidatorData[headerIndex][headerInputType] =
					value.length > 0
						? updatedHeaders[headerIndex][otherHeaderInputType].length > 0
						: updatedHeaders[headerIndex][otherHeaderInputType].length === 0;
				updatedHeadersValidatorData[headerIndex][otherHeaderInputType] =
					value.length > 0
						? updatedHeaders[headerIndex][otherHeaderInputType].length > 0
						: updatedHeaders[headerIndex][otherHeaderInputType].length === 0;
				setWebhook({ ...webhook, [inputType]: updatedHeaders });
				setValidator({
					...validator,
					[inputType]: updatedHeadersValidatorData,
				});
				break;
			default:
				break;
		}
	};
	const updateHeaders = (operation: string, index: number = 0) => {
		switch (operation) {
			case ArrayInputOperations.APPEND:
				setWebhook({
					...webhook,
					[WebhookInputDataFields.HEADERS]: [
						...(webhook?.[WebhookInputDataFields.HEADERS] || []),
						{ ...initHeadersData },
					],
				});
				setValidator({
					...validator,
					[WebhookInputDataFields.HEADERS]: [
						...(validator?.[WebhookInputDataFields.HEADERS] || []),
						{ ...initHeadersValidatorData },
					],
				});
				break;
			case ArrayInputOperations.REMOVE:
				if (webhook?.[WebhookInputDataFields.HEADERS]?.length) {
					const updatedHeaders = [...webhook[WebhookInputDataFields.HEADERS]];
					updatedHeaders.splice(index, 1);
					setWebhook({
						...webhook,
						[WebhookInputDataFields.HEADERS]: updatedHeaders,
					});
				}
				if (validator?.[WebhookInputDataFields.HEADERS]?.length) {
					const updatedHeadersData = [
						...validator[WebhookInputDataFields.HEADERS],
					];
					updatedHeadersData.splice(index, 1);
					setValidator({
						...validator,
						[WebhookInputDataFields.HEADERS]: updatedHeadersData,
					});
				}
				break;
			default:
				break;
		}
	};
	const validateData = () => {
		return (
			!loading &&
			webhook[WebhookInputDataFields.EVENT_NAME].length > 0 &&
			webhook[WebhookInputDataFields.ENDPOINT].length > 0 &&
			validator[WebhookInputDataFields.EVENT_NAME] &&
			validator[WebhookInputDataFields.ENDPOINT] &&
			!validator[WebhookInputDataFields.HEADERS].some(
				(headerData: headersValidatorDataType) =>
					!headerData.key || !headerData.value
			)
		);
	};
	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let params: any = {
			[WebhookInputDataFields.EVENT_NAME]:
				webhook[WebhookInputDataFields.EVENT_NAME],
			[WebhookInputDataFields.ENDPOINT]:
				webhook[WebhookInputDataFields.ENDPOINT],
			[WebhookInputDataFields.ENABLED]: webhook[WebhookInputDataFields.ENABLED],
			[WebhookInputDataFields.HEADERS]: {},
		};
		if (webhook[WebhookInputDataFields.HEADERS].length) {
			const headers = webhook[WebhookInputDataFields.HEADERS].reduce(
				(acc, data) => {
					return data.key ? { ...acc, [data.key]: data.value } : acc;
				},
				{}
			);
			if (Object.keys(headers).length) {
				params[WebhookInputDataFields.HEADERS] = headers;
			}
		}
		let res: any = {};
		if (
			view === UpdateWebhookModalViews.Edit &&
			selectedWebhook?.[WebhookInputDataFields.ID]
		) {
			res = await client
				.mutation(EditWebhook, {
					params: {
						...params,
						id: selectedWebhook[WebhookInputDataFields.ID],
					},
				})
				.toPromise();
		} else {
			res = await client.mutation(AddWebhook, { params }).toPromise();
		}
		if (res.error) {
			toast({
				title: capitalizeFirstLetter(res.error.message),
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});
			setLoading(false);
			return;
		} else if (res.data?._add_webhook || res.data?._update_webhook) {
			toast({
				title: capitalizeFirstLetter(
					res.data?._add_webhook?.message || res.data?._update_webhook?.message
				),
				isClosable: true,
				status: 'success',
				position: 'bottom-right',
			});
			setWebhook({
				...initWebhookData,
				[WebhookInputDataFields.HEADERS]: [{ ...initHeadersData }],
			});
			setValidator({ ...initWebhookValidatorData });
			fetchWebookData();
			return;
		}
		setLoading(false);
	};
	useEffect(() => {
		if (
			isOpen &&
			view === UpdateWebhookModalViews.Edit &&
			selectedWebhook &&
			Object.keys(selectedWebhook || {}).length
		) {
			const { headers, ...rest } = selectedWebhook;
			const headerItems = Object.entries(headers || {});
			if (headerItems.length) {
				let formattedHeadersData = headerItems.map((headerData) => {
					return {
						[WebhookInputHeaderFields.KEY]: headerData[0],
						[WebhookInputHeaderFields.VALUE]: headerData[1],
					};
				});
				setWebhook({
					...rest,
					[WebhookInputDataFields.HEADERS]: formattedHeadersData,
				});
				setValidator({
					...validator,
					[WebhookInputDataFields.HEADERS]: new Array(
						formattedHeadersData.length
					)
						.fill({})
						.map(() => ({ ...initHeadersValidatorData })),
				});
			} else {
				setWebhook({
					...rest,
					[WebhookInputDataFields.HEADERS]: [{ ...initHeadersData }],
				});
			}
		}
	}, [isOpen]);
	return (
		<>
			{view === UpdateWebhookModalViews.ADD ? (
				<Button
					leftIcon={<FaPlus />}
					colorScheme="blue"
					variant="solid"
					onClick={onOpen}
					isDisabled={false}
					size="sm"
				>
					<Center h="100%">Add Webhook</Center>{' '}
				</Button>
			) : (
				<MenuItem onClick={onOpen}>Edit</MenuItem>
			)}
			<Modal isOpen={isOpen} onClose={onClose} size="3xl">
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>
						{view === UpdateWebhookModalViews.ADD
							? 'Add New Webhook'
							: 'Edit Webhook'}
					</ModalHeader>
					<ModalCloseButton />
					<ModalBody>
						<Flex
							flexDirection="column"
							border="1px"
							borderRadius="md"
							borderColor="gray.200"
							p="5"
						>
							<Flex
								width="100%"
								justifyContent="space-between"
								alignItems="center"
								marginBottom="2%"
							>
								<Flex flex="1">Event Name</Flex>
								<Flex flex="3">
									<InputGroup size="md">
										<Input
											pr="4.5rem"
											type="text"
											placeholder="user.login"
											value={webhook[WebhookInputDataFields.EVENT_NAME]}
											isInvalid={!validator[WebhookInputDataFields.EVENT_NAME]}
											onChange={(e) =>
												inputChangehandler(
													WebhookInputDataFields.EVENT_NAME,
													e.currentTarget.value
												)
											}
										/>
									</InputGroup>
								</Flex>
							</Flex>
							<Flex
								width="100%"
								justifyContent="start"
								alignItems="center"
								marginBottom="5%"
							>
								<Flex flex="1">Endpoint</Flex>
								<Flex flex="3">
									<InputGroup size="md">
										<Input
											pr="4.5rem"
											type="text"
											placeholder="https://domain.com/webhook"
											value={webhook[WebhookInputDataFields.ENDPOINT]}
											isInvalid={!validator[WebhookInputDataFields.ENDPOINT]}
											onChange={(e) =>
												inputChangehandler(
													WebhookInputDataFields.ENDPOINT,
													e.currentTarget.value
												)
											}
										/>
									</InputGroup>
								</Flex>
							</Flex>
							<Flex
								width="100%"
								justifyContent="space-between"
								alignItems="center"
								marginBottom="5%"
							>
								<Flex flex="1">Enabled</Flex>
								<Flex w="25%" justifyContent="space-between">
									<Text h="75%" fontWeight="bold" marginRight="2">
										Off
									</Text>
									<Switch
										size="md"
										isChecked={webhook[WebhookInputDataFields.ENABLED]}
										onChange={() =>
											inputChangehandler(
												WebhookInputDataFields.ENABLED,
												!webhook[WebhookInputDataFields.ENABLED]
											)
										}
									/>
									<Text h="75%" fontWeight="bold" marginLeft="2">
										On
									</Text>
								</Flex>
							</Flex>
							<Flex
								width="100%"
								justifyContent="space-between"
								alignItems="center"
								marginBottom="2%"
							>
								<Flex>Headers</Flex>
								<Flex>
									<Button
										leftIcon={<FaPlus />}
										colorScheme="blue"
										h="1.75rem"
										size="sm"
										variant="ghost"
										paddingRight="0"
										onClick={() => updateHeaders(ArrayInputOperations.APPEND)}
									>
										Add more Headers
									</Button>
								</Flex>
							</Flex>
							<Flex flexDirection="column" maxH={220} overflowY="scroll">
								{webhook[WebhookInputDataFields.HEADERS]?.map(
									(headerData, index) => (
										<Flex
											key={`header-data-${index}`}
											justifyContent="center"
											alignItems="center"
										>
											<InputGroup size="md" marginBottom="2.5%">
												<Input
													type="text"
													placeholder="key"
													value={headerData[WebhookInputHeaderFields.KEY]}
													isInvalid={
														!validator[WebhookInputDataFields.HEADERS][index]?.[
															WebhookInputHeaderFields.KEY
														]
													}
													onChange={(e) =>
														inputChangehandler(
															WebhookInputDataFields.HEADERS,
															e.target.value,
															WebhookInputHeaderFields.KEY,
															index
														)
													}
													width="30%"
													marginRight="2%"
												/>
												<Center marginRight="2%">
													<Text fontWeight="bold">:</Text>
												</Center>
												<Input
													type="text"
													placeholder="value"
													value={headerData[WebhookInputHeaderFields.VALUE]}
													isInvalid={
														!validator[WebhookInputDataFields.HEADERS][index]?.[
															WebhookInputHeaderFields.VALUE
														]
													}
													onChange={(e) =>
														inputChangehandler(
															WebhookInputDataFields.HEADERS,
															e.target.value,
															WebhookInputHeaderFields.VALUE,
															index
														)
													}
													width="65%"
												/>
												<InputRightElement width="3rem">
													<Button
														width="6rem"
														colorScheme="blackAlpha"
														variant="ghost"
														padding="0"
														onClick={() =>
															updateHeaders(ArrayInputOperations.REMOVE, index)
														}
													>
														<FaMinusCircle />
													</Button>
												</InputRightElement>
											</InputGroup>
										</Flex>
									)
								)}
							</Flex>
						</Flex>
					</ModalBody>
					<ModalFooter>
						<Button
							colorScheme="blue"
							variant="solid"
							onClick={saveData}
							isDisabled={!validateData()}
						>
							<Center h="100%" pt="5%">
								Save
							</Center>
						</Button>
					</ModalFooter>
				</ModalContent>
			</Modal>
		</>
	);
};

export default UpdateWebhookModal;
