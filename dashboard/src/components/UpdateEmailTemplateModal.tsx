import React, { useEffect, useRef, useState } from 'react';
import {
	Button,
	Center,
	Flex,
	Input,
	InputGroup,
	MenuItem,
	Modal,
	ModalBody,
	ModalCloseButton,
	ModalContent,
	ModalFooter,
	ModalHeader,
	ModalOverlay,
	Select,
	Text,
	useDisclosure,
	useToast,
	Alert,
	AlertIcon,
	Collapse,
	Box,
	TableContainer,
	Table,
	Thead,
	Tr,
	Th,
	Tbody,
	Td,
	Code,
	Radio,
	RadioGroup,
	Stack,
	Textarea,
} from '@chakra-ui/react';
import { FaPlus, FaAngleDown, FaAngleUp } from 'react-icons/fa';
import { useClient } from 'urql';
import EmailEditor from 'react-email-editor';
import {
	UpdateModalViews,
	EmailTemplateInputDataFields,
	emailTemplateEventNames,
	emailTemplateVariables,
	EmailTemplateEditors,
} from '../constants';
import { capitalizeFirstLetter } from '../utils';
import { AddEmailTemplate, EditEmailTemplate } from '../graphql/mutation';

interface selectedEmailTemplateDataTypes {
	[EmailTemplateInputDataFields.ID]: string;
	[EmailTemplateInputDataFields.EVENT_NAME]: string;
	[EmailTemplateInputDataFields.SUBJECT]: string;
	[EmailTemplateInputDataFields.CREATED_AT]: number;
	[EmailTemplateInputDataFields.TEMPLATE]: string;
	[EmailTemplateInputDataFields.DESIGN]: string;
}

interface UpdateEmailTemplateInputPropTypes {
	view: UpdateModalViews;
	selectedTemplate?: selectedEmailTemplateDataTypes;
	fetchEmailTemplatesData: Function;
}

interface templateVariableDataTypes {
	text: string;
	value: string;
	description: string;
}

interface emailTemplateDataType {
	[EmailTemplateInputDataFields.EVENT_NAME]: string;
	[EmailTemplateInputDataFields.SUBJECT]: string;
	[EmailTemplateInputDataFields.TEMPLATE]: string;
	[EmailTemplateInputDataFields.DESIGN]: string;
}

interface validatorDataType {
	[EmailTemplateInputDataFields.SUBJECT]: boolean;
}

const initTemplateData: emailTemplateDataType = {
	[EmailTemplateInputDataFields.EVENT_NAME]: emailTemplateEventNames.Signup,
	[EmailTemplateInputDataFields.SUBJECT]: '',
	[EmailTemplateInputDataFields.TEMPLATE]: '',
	[EmailTemplateInputDataFields.DESIGN]: '',
};

const initTemplateValidatorData: validatorDataType = {
	[EmailTemplateInputDataFields.SUBJECT]: true,
};

const UpdateEmailTemplate = ({
	view,
	selectedTemplate,
	fetchEmailTemplatesData,
}: UpdateEmailTemplateInputPropTypes) => {
	const client = useClient();
	const toast = useToast();
	const emailEditorRef = useRef(null);
	const { isOpen, onOpen, onClose } = useDisclosure();
	const [loading, setLoading] = useState<boolean>(false);
	const [editor, setEditor] = useState<string>(
		EmailTemplateEditors.PLAIN_HTML_EDITOR,
	);
	const [templateVariables, setTemplateVariables] = useState<
		templateVariableDataTypes[]
	>([]);
	const [templateData, setTemplateData] = useState<emailTemplateDataType>({
		...initTemplateData,
	});
	const [validator, setValidator] = useState<validatorDataType>({
		...initTemplateValidatorData,
	});
	const [isDynamicVariableInfoOpen, setIsDynamicVariableInfoOpen] =
		useState<boolean>(false);

	const onReady = () => {
		if (selectedTemplate) {
			const { design } = selectedTemplate;
			try {
				if (design) {
					const designData = JSON.parse(design);
					// @ts-ignore
					emailEditorRef.current.editor.loadDesign(designData);
				}
			} catch (error) {
				console.error(error);
				onClose();
			}
		}
	};

	const inputChangehandler = (inputType: string, value: any) => {
		if (inputType !== EmailTemplateInputDataFields.EVENT_NAME) {
			setValidator({
				...validator,
				[inputType]: value?.trim().length,
			});
		}
		setTemplateData({ ...templateData, [inputType]: value });
	};

	const validateData = () => {
		return (
			!loading &&
			templateData[EmailTemplateInputDataFields.EVENT_NAME].length > 0 &&
			templateData[EmailTemplateInputDataFields.SUBJECT].length > 0 &&
			validator[EmailTemplateInputDataFields.SUBJECT]
		);
	};

	const updateTemplate = async (params: emailTemplateDataType) => {
		let res: any = {};
		if (
			view === UpdateModalViews.Edit &&
			selectedTemplate?.[EmailTemplateInputDataFields.ID]
		) {
			res = await client
				.mutation(EditEmailTemplate, {
					params: {
						...params,
						id: selectedTemplate[EmailTemplateInputDataFields.ID],
					},
				})
				.toPromise();
		} else {
			res = await client.mutation(AddEmailTemplate, { params }).toPromise();
		}
		setLoading(false);
		if (res.error) {
			toast({
				title: capitalizeFirstLetter(res.error.message),
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});
		} else if (
			res.data?._add_email_template ||
			res.data?._update_email_template
		) {
			toast({
				title: capitalizeFirstLetter(
					res.data?._add_email_template?.message ||
						res.data?._update_email_template?.message,
				),
				isClosable: true,
				status: 'success',
				position: 'bottom-right',
			});
			setTemplateData({
				...initTemplateData,
			});
			setValidator({ ...initTemplateValidatorData });
			fetchEmailTemplatesData();
		}
	};

	const saveData = async () => {
		if (!validateData()) return;
		setLoading(true);
		let params: emailTemplateDataType = {
			[EmailTemplateInputDataFields.EVENT_NAME]:
				templateData[EmailTemplateInputDataFields.EVENT_NAME],
			[EmailTemplateInputDataFields.SUBJECT]:
				templateData[EmailTemplateInputDataFields.SUBJECT],
			[EmailTemplateInputDataFields.TEMPLATE]:
				templateData[EmailTemplateInputDataFields.TEMPLATE],
			[EmailTemplateInputDataFields.DESIGN]: '',
		};
		if (editor === EmailTemplateEditors.UNLAYER_EDITOR) {
			// @ts-ignore
			await emailEditorRef.current.editor.exportHtml(async (data) => {
				const { design, html } = data;
				if (!html || !design) {
					setLoading(false);
					return;
				}
				params = {
					...params,
					[EmailTemplateInputDataFields.TEMPLATE]: html.trim(),
					[EmailTemplateInputDataFields.DESIGN]: JSON.stringify(design),
				};
				await updateTemplate(params);
			});
		} else {
			await updateTemplate(params);
		}
		view === UpdateModalViews.ADD && onClose();
	};

	const resetData = () => {
		if (selectedTemplate) {
			setTemplateData(selectedTemplate);
		} else {
			setTemplateData({ ...initTemplateData });
		}
	};

	// set template data if edit modal is open
	useEffect(() => {
		if (
			isOpen &&
			view === UpdateModalViews.Edit &&
			selectedTemplate &&
			Object.keys(selectedTemplate || {}).length
		) {
			const { id, created_at, ...rest } = selectedTemplate;
			setTemplateData(rest);
		}
	}, [isOpen]);

	// set template variables
	useEffect(() => {
		const updatedTemplateVariables = Object.entries(
			emailTemplateVariables,
		).reduce((acc, [key, val]): any => {
			if (
				(templateData[EmailTemplateInputDataFields.EVENT_NAME] !==
					emailTemplateEventNames['Verify Otp'] &&
					val === emailTemplateVariables.otp) ||
				(templateData[EmailTemplateInputDataFields.EVENT_NAME] ===
					emailTemplateEventNames['Verify Otp'] &&
					val === emailTemplateVariables.verification_url)
			) {
				return acc;
			}
			return [
				...acc,
				{
					text: key,
					value: val.value,
					description: val.description,
				},
			];
		}, []);
		setTemplateVariables(updatedTemplateVariables);
	}, [templateData[EmailTemplateInputDataFields.EVENT_NAME]]);

	// change editor
	useEffect(() => {
		if (isOpen && selectedTemplate) {
			const { design } = selectedTemplate;
			if (design) {
				setEditor(EmailTemplateEditors.UNLAYER_EDITOR);
			} else {
				setEditor(EmailTemplateEditors.PLAIN_HTML_EDITOR);
			}
		}
	}, [isOpen, selectedTemplate]);

	// reset fields when editor is changed
	useEffect(() => {
		if (selectedTemplate?.design) {
			if (editor === EmailTemplateEditors.UNLAYER_EDITOR) {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: selectedTemplate.template,
					[EmailTemplateInputDataFields.DESIGN]: selectedTemplate.design,
				});
			} else {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: '',
					[EmailTemplateInputDataFields.DESIGN]: '',
				});
			}
		} else if (selectedTemplate?.template) {
			if (editor === EmailTemplateEditors.UNLAYER_EDITOR) {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: '',
					[EmailTemplateInputDataFields.DESIGN]: '',
				});
			} else {
				setTemplateData({
					...templateData,
					[EmailTemplateInputDataFields.TEMPLATE]: selectedTemplate?.template,
					[EmailTemplateInputDataFields.DESIGN]: '',
				});
			}
		}
	}, [editor]);

	return (
		<>
			{view === UpdateModalViews.ADD ? (
				<Button
					leftIcon={<FaPlus />}
					colorScheme="blue"
					variant="solid"
					onClick={onOpen}
					isDisabled={false}
					size="sm"
				>
					<Center h="100%">Add Template</Center>{' '}
				</Button>
			) : (
				<MenuItem onClick={onOpen}>Edit</MenuItem>
			)}
			<Modal
				isOpen={isOpen}
				onClose={() => {
					resetData();
					onClose();
				}}
				size="6xl"
			>
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>
						{view === UpdateModalViews.ADD
							? 'Add New Email Template'
							: 'Edit Email Template'}
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
							<Alert
								status="info"
								onClick={() =>
									setIsDynamicVariableInfoOpen(!isDynamicVariableInfoOpen)
								}
								borderRadius="5"
								marginBottom={5}
								cursor="pointer"
								fontSize="sm"
							>
								<AlertIcon />
								<Flex
									width="100%"
									justifyContent="space-between"
									alignItems="center"
								>
									<Box width="85%">
										<b>Note:</b> You can add set of dynamic variables to subject
										and email body. Click here to see the set of dynamic
										variables.
									</Box>
									{isDynamicVariableInfoOpen ? <FaAngleUp /> : <FaAngleDown />}
								</Flex>
							</Alert>
							<Collapse
								style={{
									width: '100%',
								}}
								in={isDynamicVariableInfoOpen}
							>
								<TableContainer
									background="gray.100"
									borderRadius={5}
									height={200}
									width="100%"
									overflowY="auto"
									overflowWrap="break-word"
								>
									<Table variant="simple">
										<Thead>
											<Tr>
												<Th>Variable</Th>
												<Th>Description</Th>
											</Tr>
										</Thead>
										<Tbody>
											{templateVariables.map((i) => (
												<Tr key={i.text}>
													<Td>
														<Code fontSize="sm">{`{{.${i.text}}}`}</Code>
													</Td>
													<Td>
														<Text
															size="sm"
															fontSize="sm"
															overflowWrap="break-word"
															width="100%"
														>
															{i.description}
														</Text>
													</Td>
												</Tr>
											))}
										</Tbody>
									</Table>
								</TableContainer>
							</Collapse>
							<Flex
								width="100%"
								justifyContent="space-between"
								alignItems="center"
								marginBottom="2%"
							>
								<Flex flex="1">Event Name</Flex>
								<Flex flex="3">
									<Select
										size="md"
										value={
											templateData[EmailTemplateInputDataFields.EVENT_NAME]
										}
										onChange={(e) =>
											inputChangehandler(
												EmailTemplateInputDataFields.EVENT_NAME,
												e.currentTarget.value,
											)
										}
									>
										{Object.entries(emailTemplateEventNames).map(
											([key, value]: any) => (
												<option value={value} key={key}>
													{key}
												</option>
											),
										)}
									</Select>
								</Flex>
							</Flex>
							<Flex
								width="100%"
								justifyContent="start"
								alignItems="center"
								marginBottom="2%"
							>
								<Flex flex="1">Subject</Flex>
								<Flex flex="3">
									<InputGroup size="md">
										<Input
											pr="4.5rem"
											type="text"
											placeholder="Subject Line"
											value={templateData[EmailTemplateInputDataFields.SUBJECT]}
											isInvalid={
												!validator[EmailTemplateInputDataFields.SUBJECT]
											}
											onChange={(e) =>
												inputChangehandler(
													EmailTemplateInputDataFields.SUBJECT,
													e.currentTarget.value,
												)
											}
										/>
									</InputGroup>
								</Flex>
							</Flex>
							<Flex
								width="100%"
								justifyContent="flex-start"
								alignItems="center"
								marginBottom="2%"
							>
								<Flex flex="1">Template Body</Flex>
								<Flex flex="3">
									<RadioGroup
										onChange={(value) => setEditor(value)}
										value={editor}
									>
										<Stack direction="row" spacing="50px">
											<Radio value={EmailTemplateEditors.PLAIN_HTML_EDITOR}>
												Plain HTML
											</Radio>
											<Radio value={EmailTemplateEditors.UNLAYER_EDITOR}>
												Unlayer Editor
											</Radio>
										</Stack>
									</RadioGroup>
								</Flex>
							</Flex>
							<Flex
								width="100%"
								justifyContent="flex-start"
								alignItems="center"
								border="1px solid"
								borderColor="gray.200"
							>
								{editor === EmailTemplateEditors.UNLAYER_EDITOR ? (
									<EmailEditor ref={emailEditorRef} onReady={onReady} />
								) : (
									<Textarea
										value={templateData.template}
										onChange={(e) => {
											setTemplateData({
												...templateData,
												[EmailTemplateInputDataFields.TEMPLATE]: e.target.value,
											});
										}}
										placeholder="Template HTML"
										border="0"
										height="500px"
									/>
								)}
							</Flex>
						</Flex>
					</ModalBody>
					<ModalFooter>
						<Button
							variant="outline"
							onClick={resetData}
							isDisabled={loading}
							marginRight="5"
						>
							Reset
						</Button>
						<Button
							colorScheme="blue"
							variant="solid"
							isLoading={loading}
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

export default UpdateEmailTemplate;
