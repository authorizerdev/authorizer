import React, { useState, useCallback, useEffect } from 'react';
import {
	Button,
	Center,
	Flex,
	Modal,
	ModalBody,
	ModalCloseButton,
	ModalContent,
	ModalFooter,
	ModalHeader,
	ModalOverlay,
	useDisclosure,
	useToast,
	Tabs,
	TabList,
	Tab,
	TabPanels,
	TabPanel,
	InputGroup,
	Input,
	InputRightElement,
	Text,
	Link,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaUserPlus, FaMinusCircle, FaPlus, FaUpload } from 'react-icons/fa';
import { useDropzone } from 'react-dropzone';
import { escape } from 'lodash';
import { validateEmail, validateURI } from '../utils';
import { UpdateUser } from '../graphql/mutation';
import { ArrayInputOperations, csvDemoData } from '../constants';
import parseCSV from '../utils/parseCSV';

interface stateDataTypes {
	value: string;
	isInvalid: boolean;
}

const InviteMembersModal = ({ disabled = true }: { disabled: boolean }) => {
	const client = useClient();
	const toast = useToast();
	const { isOpen, onOpen, onClose } = useDisclosure();
	const [tabIndex, setTabIndex] = useState<number>(0);
	const [redirectURI, setRedirectURI] = useState<stateDataTypes>({
		value: '',
		isInvalid: false,
	});
	const [emails, setEmails] = useState<stateDataTypes[]>([
		{
			value: '',
			isInvalid: false,
		},
	]);
	const [disableSendButton, setDisableSendButton] = useState<boolean>(false);
	useEffect(() => {
		if (redirectURI.isInvalid) {
			setDisableSendButton(true);
		} else if (emails.some((emailData) => emailData.isInvalid)) {
			setDisableSendButton(true);
		} else {
			setDisableSendButton(false);
		}
	}, [redirectURI, emails]);
	const sendInviteHandler = async () => {
		onClose();
	};
	const updateEmailListHandler = (operation: string, index: number = 0) => {
		switch (operation) {
			case ArrayInputOperations.APPEND:
				setEmails([
					...emails,
					{
						value: '',
						isInvalid: false,
					},
				]);
				break;
			case ArrayInputOperations.REMOVE:
				const updatedEmailList = [...emails];
				updatedEmailList.splice(index, 1);
				setEmails(updatedEmailList);
				break;
			default:
				break;
		}
	};
	const inputChangeHandler = (value: string, index: number) => {
		const updatedEmailList = [...emails];
		updatedEmailList[index].value = value;
		updatedEmailList[index].isInvalid = !validateEmail(value);
		setEmails(updatedEmailList);
	};
	const changeTabsHandler = (index: number) => {
		setTabIndex(index);
	};
	const onDrop = useCallback(async (acceptedFiles) => {
		const result = await parseCSV(acceptedFiles[0], ',');
		setEmails(result);
		changeTabsHandler(0);
	}, []);
	const setRedirectURIHandler = (value: string) => {
		const updatedRedirectURI: stateDataTypes = {
			value: '',
			isInvalid: false,
		};
		updatedRedirectURI.value = value;
		updatedRedirectURI.isInvalid = !validateURI(value);
		setRedirectURI(updatedRedirectURI);
	};
	const { getRootProps, getInputProps, isDragActive } = useDropzone({
		onDrop,
		accept: 'text/csv',
	});
	return (
		<>
			<Button
				leftIcon={<FaUserPlus />}
				colorScheme="blue"
				variant="solid"
				onClick={onOpen}
				isDisabled={disabled}
				size="sm"
			>
				<Center h="100%" pt="5%">
					Invite Members
				</Center>
			</Button>
			<Modal isOpen={isOpen} onClose={onClose} size="xl">
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>Invite Members</ModalHeader>
					<ModalCloseButton />
					<ModalBody>
						<Tabs
							isFitted
							variant="enclosed"
							index={tabIndex}
							onChange={changeTabsHandler}
						>
							<TabList>
								<Tab>Enter emails</Tab>
								<Tab>Upload CSV</Tab>
							</TabList>
							<TabPanels
								border="1px"
								borderTop="0"
								borderBottomRadius="5px"
								borderColor="inherit"
							>
								<TabPanel>
									<Flex flexDirection="column">
										<Flex
											width="100%"
											justifyContent="start"
											alignItems="center"
											marginBottom="2%"
										>
											<Flex marginLeft="2.5%">Redirect URI</Flex>
										</Flex>
										<Flex
											width="100%"
											justifyContent="space-between"
											alignItems="center"
											marginBottom="2%"
										>
											<InputGroup size="md" marginBottom="2.5%">
												<Input
													pr="4.5rem"
													type="text"
													placeholder="https://domain.com/sign-up"
													value={redirectURI.value}
													isInvalid={redirectURI.isInvalid}
													onChange={(e) =>
														setRedirectURIHandler(e.currentTarget.value)
													}
												/>
											</InputGroup>
										</Flex>
										<Flex
											width="100%"
											justifyContent="space-between"
											alignItems="center"
											marginBottom="2%"
										>
											<Flex marginLeft="2.5%">Emails</Flex>
											<Flex>
												<Button
													leftIcon={<FaPlus />}
													colorScheme="blue"
													h="1.75rem"
													size="sm"
													variant="ghost"
													onClick={() =>
														updateEmailListHandler(ArrayInputOperations.APPEND)
													}
												>
													Add more emails
												</Button>
											</Flex>
										</Flex>
										{emails.map((emailData, index) => (
											<Flex
												key={`email-data-${index}`}
												justifyContent="center"
												alignItems="center"
											>
												<InputGroup size="md" marginBottom="2.5%">
													<Input
														pr="4.5rem"
														type="text"
														placeholder="name@domain.com"
														value={emailData.value}
														isInvalid={emailData.isInvalid}
														onChange={(e) =>
															inputChangeHandler(e.currentTarget.value, index)
														}
													/>
													<InputRightElement width="3rem">
														<Button
															h="1.75rem"
															size="sm"
															colorScheme="blackAlpha"
															variant="ghost"
															onClick={() =>
																updateEmailListHandler(
																	ArrayInputOperations.REMOVE,
																	index
																)
															}
														>
															<FaMinusCircle />
														</Button>
													</InputRightElement>
												</InputGroup>
											</Flex>
										))}
									</Flex>
								</TabPanel>
								<TabPanel>
									<Flex
										justify="center"
										align="center"
										textAlign="center"
										bg="#f0f0f0"
										h={231}
										p={50}
										m={2}
										borderRadius={5}
										{...getRootProps()}
									>
										<input {...getInputProps()} />
										{isDragActive ? (
											<Text>Drop the files here...</Text>
										) : (
											<Flex
												flexDirection="column"
												justifyContent="center"
												alignItems="center"
											>
												<Center boxSize="20" color="blackAlpha.500">
													<FaUpload fontSize="40" />
												</Center>
												<Text>
													Drag 'n' drop the csv file here, or click to select.
												</Text>
												<Text size="xs">
													Download{' '}
													<Link
														href={`data:text/csv;charset=utf-8,${escape(
															csvDemoData
														)}`}
														download="sample.csv"
														color="blue.600"
														onClick={(e) => e.stopPropagation()}
													>
														{' '}
														sample.csv
													</Link>{' '}
													and modify it.{' '}
												</Text>
											</Flex>
										)}
									</Flex>
								</TabPanel>
							</TabPanels>
						</Tabs>
					</ModalBody>
					<ModalFooter>
						<Button
							colorScheme="blue"
							variant="solid"
							onClick={sendInviteHandler}
							isDisabled={disableSendButton}
						>
							<Center h="100%" pt="5%">
								Send
							</Center>
						</Button>
					</ModalFooter>
				</ModalContent>
			</Modal>
		</>
	);
};

export default InviteMembersModal;
