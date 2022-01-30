import React from 'react';
import {
	Box,
	Flex,
	Input,
	Center,
	InputGroup,
	InputRightElement,
	Tag,
	TagLabel,
	TagRightIcon,
	Select,
	Textarea,
	Switch,
	Code,
} from '@chakra-ui/react';
import {
	FaRegClone,
	FaRegEye,
	FaRegEyeSlash,
	FaPlus,
	FaTimes,
} from 'react-icons/fa';
import {
	ArrayInputOperations,
	ArrayInputType,
	SelectInputType,
	HiddenInputType,
	TextInputType,
	TextAreaInputType,
	SwitchInputType,
	DateInputType,
} from '../constants';
import { copyTextToClipboard } from '../utils';

const InputField = ({
	inputType,
	variables,
	setVariables,
	fieldVisibility,
	setFieldVisibility,
	...downshiftProps
}: any) => {
	const props = {
		size: 'sm',
		...downshiftProps,
	};
	const [inputFieldVisibility, setInputFieldVisibility] = React.useState<
		Record<string, boolean>
	>({
		ROLES: false,
		DEFAULT_ROLES: false,
		PROTECTED_ROLES: false,
		ALLOWED_ORIGINS: false,
		roles: false,
	});
	const [inputData, setInputData] = React.useState<Record<string, string>>({
		ROLES: '',
		DEFAULT_ROLES: '',
		PROTECTED_ROLES: '',
		ALLOWED_ORIGINS: '',
		roles: '',
	});
	const updateInputHandler = (
		type: string,
		operation: any,
		role: string = ''
	) => {
		if (operation === ArrayInputOperations.APPEND) {
			if (inputData[type] !== '') {
				setVariables({
					...variables,
					[type]: [...variables[type], inputData[type]],
				});
				setInputData({ ...inputData, [type]: '' });
			}
			setInputFieldVisibility({ ...inputFieldVisibility, [type]: false });
		}
		if (operation === ArrayInputOperations.REMOVE) {
			let updatedEnvVars = variables[type].filter(
				(item: string) => item !== role
			);
			setVariables({
				...variables,
				[type]: updatedEnvVars,
			});
		}
	};
	if (Object.values(TextInputType).includes(inputType)) {
		return (
			<InputGroup size="sm">
				<Input
					{...props}
					value={variables[inputType] ? variables[inputType] : ''}
					onChange={(
						event: Event & {
							target: HTMLInputElement;
						}
					) =>
						setVariables({
							...variables,
							[inputType]: event.target.value,
						})
					}
				/>
				<InputRightElement
					children={<FaRegClone color="#bfbfbf" />}
					cursor="pointer"
					onClick={() => copyTextToClipboard(variables[inputType])}
				/>
			</InputGroup>
		);
	}
	if (Object.values(HiddenInputType).includes(inputType)) {
		return (
			<InputGroup size="sm">
				<Input
					{...props}
					value={variables[inputType]}
					onChange={(
						event: Event & {
							target: HTMLInputElement;
						}
					) =>
						setVariables({
							...variables,
							[inputType]: event.target.value,
						})
					}
					type={!fieldVisibility[inputType] ? 'password' : 'text'}
				/>
				<InputRightElement
					right="15px"
					children={
						<Flex>
							{fieldVisibility[inputType] ? (
								<Center
									w="25px"
									margin="0 1.5%"
									cursor="pointer"
									onClick={() =>
										setFieldVisibility({
											...fieldVisibility,
											[inputType]: false,
										})
									}
								>
									<FaRegEyeSlash color="#bfbfbf" />
								</Center>
							) : (
								<Center
									w="25px"
									margin="0 1.5%"
									cursor="pointer"
									onClick={() =>
										setFieldVisibility({
											...fieldVisibility,
											[inputType]: true,
										})
									}
								>
									<FaRegEye color="#bfbfbf" />
								</Center>
							)}
							<Center
								w="25px"
								margin="0 1.5%"
								cursor="pointer"
								onClick={() => copyTextToClipboard(variables[inputType])}
							>
								<FaRegClone color="#bfbfbf" />
							</Center>
						</Flex>
					}
				/>
			</InputGroup>
		);
	}
	if (Object.values(ArrayInputType).includes(inputType)) {
		return (
			<Flex
				border="1px solid #e2e8f0"
				w="100%"
				paddingTop="0.5%"
				overflowX="scroll"
				overflowY="hidden"
				justifyContent="start"
				alignItems="center"
			>
				{variables[inputType].map((role: string, index: number) => (
					<Box key={index} margin="0.5%" role="group">
						<Tag
							size="sm"
							variant="outline"
							colorScheme="gray"
							minW="fit-content"
						>
							<TagLabel cursor="default">{role}</TagLabel>
							<TagRightIcon
								boxSize="12px"
								as={FaTimes}
								display="none"
								cursor="pointer"
								_groupHover={{ display: 'block' }}
								onClick={() =>
									updateInputHandler(
										inputType,
										ArrayInputOperations.REMOVE,
										role
									)
								}
							/>
						</Tag>
					</Box>
				))}
				{inputFieldVisibility[inputType] ? (
					<Box ml="1%" mb="0.75%">
						<Input
							type="text"
							size="xs"
							minW="150px"
							placeholder="add a new value"
							value={inputData[inputType]}
							onChange={(e: any) => {
								setInputData({ ...inputData, [inputType]: e.target.value });
							}}
							onBlur={() =>
								updateInputHandler(inputType, ArrayInputOperations.APPEND)
							}
							onKeyPress={(event) => {
								if (event.key === 'Enter') {
									updateInputHandler(inputType, ArrayInputOperations.APPEND);
								}
							}}
						/>
					</Box>
				) : (
					<Box
						marginLeft="0.5%"
						cursor="pointer"
						onClick={() =>
							setInputFieldVisibility({
								...inputFieldVisibility,
								[inputType]: true,
							})
						}
					>
						<Tag
							size="sm"
							variant="outline"
							colorScheme="gray"
							minW="fit-content"
						>
							<FaPlus />
						</Tag>
					</Box>
				)}
			</Flex>
		);
	}
	if (Object.values(SelectInputType).includes(inputType)) {
		if (inputType === SelectInputType.JWT_TYPE) {
			return (
				<Select size="sm" {...props}>
					{[variables[inputType]].map((value: string) => (
						<option value="value" key={value}>
							{value}
						</option>
					))}
				</Select>
			);
		}
		const { options, ...rest } = props;
		return (
			<Select
				size="sm"
				{...rest}
				value={variables[inputType] ? variables[inputType] : ''}
				onChange={(e) =>
					setVariables({ ...variables, [inputType]: e.target.value })
				}
			>
				{Object.entries(options).map(([key, value]: any) => (
					<option value={value} key={key}>
						{key}
					</option>
				))}
			</Select>
		);
	}
	if (Object.values(TextAreaInputType).includes(inputType)) {
		return (
			<Textarea
				{...props}
				size="lg"
				value={inputData[inputType]}
				onChange={(e: any) => {
					setInputData({ ...inputData, [inputType]: e.target.value });
				}}
			/>
		);
	}
	if (Object.values(SwitchInputType).includes(inputType)) {
		return (
			<Flex w="25%" justifyContent="space-between">
				<Code h="75%">Off</Code>
				<Switch
					size="md"
					isChecked={variables[inputType]}
					onChange={() => {
						setVariables({
							...variables,
							[inputType]: !variables[inputType],
						});
					}}
				/>
				<Code h="75%">On</Code>
			</Flex>
		);
	}
	if (Object.values(DateInputType).includes(inputType)) {
		return (
			<Flex border="1px solid #e2e8f0" w="100%" h="33px" padding="1%">
				<input
					type="date"
					style={{ width: '100%', paddingLeft: '2.5%' }}
					value={variables[inputType] ? variables[inputType] : ''}
					onChange={(e) =>
						setVariables({ ...variables, [inputType]: e.target.value })
					}
				/>
			</Flex>
		);
	}
	return null;
};

export default InputField;
