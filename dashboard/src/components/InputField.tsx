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
} from '../constants';
import { copyTextToClipboard } from '../utils';

const InputField = ({
	inputType,
	envVariables,
	setEnvVariables,
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
	});
	const [inputData, setInputData] = React.useState<Record<string, string>>({
		ROLES: '',
		DEFAULT_ROLES: '',
		PROTECTED_ROLES: '',
		ALLOWED_ORIGINS: '',
	});
	const updateInputHandler = (
		type: string,
		operation: any,
		role: string = ''
	) => {
		if (operation === ArrayInputOperations.APPEND) {
			if (inputData[type] !== '') {
				setEnvVariables({
					...envVariables,
					[type]: [...envVariables[type], inputData[type]],
				});
				setInputData({ ...inputData, [type]: '' });
			}
			setInputFieldVisibility({ ...inputFieldVisibility, [type]: false });
		}
		if (operation === ArrayInputOperations.REMOVE) {
			let updatedEnvVars = envVariables[type].filter(
				(item: string) => item !== role
			);
			setEnvVariables({
				...envVariables,
				[type]: updatedEnvVars,
			});
		}
	};
	if (Object.values(TextInputType).includes(inputType)) {
		return (
			<InputGroup size="sm">
				<Input
					{...props}
					value={envVariables[inputType]}
					onChange={(
						event: Event & {
							target: HTMLInputElement;
						}
					) =>
						setEnvVariables({
							...envVariables,
							[inputType]: event.target.value,
						})
					}
				/>
				<InputRightElement
					children={<FaRegClone color="#bfbfbf" />}
					cursor="pointer"
					onClick={() => copyTextToClipboard(envVariables[inputType])}
				/>
			</InputGroup>
		);
	}
	if (Object.values(HiddenInputType).includes(inputType)) {
		return (
			<InputGroup size="sm">
				<Input
					{...props}
					value={envVariables[inputType]}
					onChange={(
						event: Event & {
							target: HTMLInputElement;
						}
					) =>
						setEnvVariables({
							...envVariables,
							[inputType]: event.target.value,
						})
					}
					type={!fieldVisibility[inputType] ? 'password' : 'text'}
				/>
				<InputRightElement
					right="15px"
					children={
						<Flex bgColor="white">
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
								onClick={() => copyTextToClipboard(envVariables[inputType])}
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
				h="45px"
				wrap="wrap"
				overflow="scroll"
				padding="1%"
			>
				{envVariables[inputType].map((role: string, index: number) => (
					<Box key={index} margin="1" role="group">
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
					<Box ml="1.15%">
						<Input
							type="text"
							size="xs"
							placeholder="add a new value"
							value={inputData[inputType]}
							onChange={(e: any) => {
								setInputData({ ...inputData, [inputType]: e.target.value });
							}}
							onBlur={() =>
								updateInputHandler(inputType, ArrayInputOperations.APPEND)
							}
						/>
					</Box>
				) : (
					<Box
						margin="1"
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
		return (
			<Select size="sm" {...props}>
				{[envVariables[inputType]].map((value: string) => (
					<option value="value">{value}</option>
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
			<Switch
				size="md"
				isChecked={envVariables[inputType]}
				onChange={() => {
					setEnvVariables({
						...envVariables,
						[inputType]: !envVariables[inputType],
					});
				}}
			/>
		);
	}
	return null;
};

export default InputField;
