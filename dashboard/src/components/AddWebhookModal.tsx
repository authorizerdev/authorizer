import React, { useState } from 'react';
import {
	Button,
	Center,
	Flex,
	Input,
	InputGroup,
	InputRightElement,
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
import { ArrayInputOperations } from '../constants';
import {
	capitalizeFirstLetter,
	validateEventName,
	validateURI,
} from '../utils';
import { AddWebhook } from '../graphql/mutation';

enum INPUT_FIELDS {
	EVENT_NAME = 'event_name',
	ENDPOINT = 'endpoint',
	ENABLED = 'enabled',
	HEADERS = 'headers',
}

enum HEADER_FIELDS {
	KEY = 'key',
	VALUE = 'value',
}

interface headersDataType {
	[HEADER_FIELDS.KEY]: string;
	[HEADER_FIELDS.VALUE]: string;
}

interface headersValidatorDataType {
	[HEADER_FIELDS.KEY]: boolean;
	[HEADER_FIELDS.VALUE]: boolean;
}

const initHeadersData: headersDataType = {
	[HEADER_FIELDS.KEY]: '',
	[HEADER_FIELDS.VALUE]: '',
};

const initHeadersValidatorData: headersValidatorDataType = {
	[HEADER_FIELDS.KEY]: true,
	[HEADER_FIELDS.VALUE]: true,
};

interface webhookDataType {
	[INPUT_FIELDS.EVENT_NAME]: string;
	[INPUT_FIELDS.ENDPOINT]: string;
	[INPUT_FIELDS.ENABLED]: boolean;
	[INPUT_FIELDS.HEADERS]: headersDataType[];
}

interface validatorDataType {
	[INPUT_FIELDS.EVENT_NAME]: boolean;
	[INPUT_FIELDS.ENDPOINT]: boolean;
	[INPUT_FIELDS.HEADERS]: headersValidatorDataType[];
}

const initWebhookData: webhookDataType = {
	[INPUT_FIELDS.EVENT_NAME]: '',
	[INPUT_FIELDS.ENDPOINT]: '',
	[INPUT_FIELDS.ENABLED]: false,
	[INPUT_FIELDS.HEADERS]: [{ ...initHeadersData }],
};

const initWebhookValidatorData: validatorDataType = {
	[INPUT_FIELDS.EVENT_NAME]: true,
	[INPUT_FIELDS.ENDPOINT]: true,
	[INPUT_FIELDS.HEADERS]: [{ ...initHeadersValidatorData }],
};

const AddWebhookModal = () => {
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
		headerInputType: string = HEADER_FIELDS.KEY,
		headerIndex: number = 0
	) => {
		switch (inputType) {
			case INPUT_FIELDS.EVENT_NAME:
				setWebhook({ ...webhook, [inputType]: value });
				setValidator({
					...validator,
					[INPUT_FIELDS.EVENT_NAME]: validateEventName(value),
				});
				break;
			case INPUT_FIELDS.ENDPOINT:
				setWebhook({ ...webhook, [inputType]: value });
				setValidator({
					...validator,
					[INPUT_FIELDS.ENDPOINT]: validateURI(value),
				});
				break;
			case INPUT_FIELDS.ENABLED:
				setWebhook({ ...webhook, [inputType]: value });
				break;
			case INPUT_FIELDS.HEADERS:
				const updatedHeaders: any = [...webhook[INPUT_FIELDS.HEADERS]];
				const updatedHeadersValidatorData: any = [
					...validator[INPUT_FIELDS.HEADERS],
				];
				const otherHeaderInputType =
					headerInputType === HEADER_FIELDS.KEY
						? HEADER_FIELDS.VALUE
						: HEADER_FIELDS.KEY;
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
					[INPUT_FIELDS.HEADERS]: [
						...(webhook?.[INPUT_FIELDS.HEADERS] || []),
						{ ...initHeadersData },
					],
				});
				setValidator({
					...validator,
					[INPUT_FIELDS.HEADERS]: [
						...(validator?.[INPUT_FIELDS.HEADERS] || []),
						{ ...initHeadersValidatorData },
					],
				});
				break;
			case ArrayInputOperations.REMOVE:
				if (webhook?.[INPUT_FIELDS.HEADERS]?.length) {
					const updatedHeaders = [...webhook[INPUT_FIELDS.HEADERS]];
					updatedHeaders.splice(index, 1);
					setWebhook({
						...webhook,
						[INPUT_FIELDS.HEADERS]: updatedHeaders,
					});
				}
				if (validator?.[INPUT_FIELDS.HEADERS]?.length) {
					const updatedHeadersData = [...validator[INPUT_FIELDS.HEADERS]];
					updatedHeadersData.splice(index, 1);
					setValidator({
						...validator,
						[INPUT_FIELDS.HEADERS]: updatedHeadersData,
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
			webhook[INPUT_FIELDS.EVENT_NAME].length > 0 &&
			webhook[INPUT_FIELDS.ENDPOINT].length > 0 &&
			validator[INPUT_FIELDS.EVENT_NAME] &&
			validator[INPUT_FIELDS.ENDPOINT] &&
			!validator[INPUT_FIELDS.HEADERS].some(
				(headerData: headersValidatorDataType) =>
					!headerData.key || !headerData.value
			)
		);
	};
	const saveData = async () => {
		if (!validateData()) return;
		let params: any = {
			[INPUT_FIELDS.EVENT_NAME]: webhook[INPUT_FIELDS.EVENT_NAME],
			[INPUT_FIELDS.ENDPOINT]: webhook[INPUT_FIELDS.ENDPOINT],
			[INPUT_FIELDS.ENABLED]: webhook[INPUT_FIELDS.ENABLED],
		};
		if (webhook[INPUT_FIELDS.HEADERS].length > 0) {
			const headers = webhook[INPUT_FIELDS.HEADERS].reduce((acc, data) => {
				return { ...acc, [data.key]: data.value };
			}, {});
			params[INPUT_FIELDS.HEADERS] = headers;
		}
		const res = await client.mutation(AddWebhook, { params }).toPromise();
		if (res.error) {
			toast({
				title: capitalizeFirstLetter(res.error.message),
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});
			return;
		} else if (res.data?._add_webhook) {
			toast({
				title: capitalizeFirstLetter(res.data?._add_webhook.message),
				isClosable: true,
				status: 'success',
				position: 'bottom-right',
			});
			setWebhook({ ...initWebhookData });
			onClose();
		}
	};
	return (
		<>
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
			<Modal isOpen={isOpen} onClose={onClose} size="3xl">
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>Add New Webhook</ModalHeader>
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
											value={webhook[INPUT_FIELDS.EVENT_NAME]}
											isInvalid={!validator[INPUT_FIELDS.EVENT_NAME]}
											onChange={(e) =>
												inputChangehandler(
													INPUT_FIELDS.EVENT_NAME,
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
											value={webhook[INPUT_FIELDS.ENDPOINT]}
											isInvalid={!validator[INPUT_FIELDS.ENDPOINT]}
											onChange={(e) =>
												inputChangehandler(
													INPUT_FIELDS.ENDPOINT,
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
										isChecked={webhook[INPUT_FIELDS.ENABLED]}
										onChange={() =>
											inputChangehandler(
												INPUT_FIELDS.ENABLED,
												!webhook[INPUT_FIELDS.ENABLED]
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
								{webhook[INPUT_FIELDS.HEADERS]?.map((headerData, index) => (
									<Flex
										key={`header-data-${index}`}
										justifyContent="center"
										alignItems="center"
									>
										<InputGroup size="md" marginBottom="2.5%">
											<Input
												type="text"
												placeholder="key"
												value={headerData[HEADER_FIELDS.KEY]}
												isInvalid={
													!validator[INPUT_FIELDS.HEADERS][index][
														HEADER_FIELDS.KEY
													]
												}
												onChange={(e) =>
													inputChangehandler(
														INPUT_FIELDS.HEADERS,
														e.target.value,
														HEADER_FIELDS.KEY,
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
												value={headerData[HEADER_FIELDS.VALUE]}
												isInvalid={
													!validator[INPUT_FIELDS.HEADERS][index][
														HEADER_FIELDS.VALUE
													]
												}
												onChange={(e) =>
													inputChangehandler(
														INPUT_FIELDS.HEADERS,
														e.target.value,
														HEADER_FIELDS.VALUE,
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
								))}
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

export default AddWebhookModal;
