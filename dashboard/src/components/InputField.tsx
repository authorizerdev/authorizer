import React, { useState } from 'react';
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
	Text,
	MenuButton,
	MenuList,
	MenuItemOption,
	MenuOptionGroup,
	Button,
	Menu,
} from '@chakra-ui/react';
import {
	FaRegClone,
	FaRegEye,
	FaRegEyeSlash,
	FaPlus,
	FaTimes,
	FaAngleDown,
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
	MultiSelectInputType,
} from '../constants';
import { copyTextToClipboard } from '../utils';

const InputField = ({
	inputType,
	variables,
	setVariables,
	fieldVisibility,
	setFieldVisibility,
	availableRoles,
	...downshiftProps
}: any) => {
	const props = {
		size: 'sm',
		...downshiftProps,
	};
	const [availableUserRoles, setAvailableUserRoles] =
		useState<string[]>(availableRoles);
	const [inputFieldVisibility, setInputFieldVisibility] = useState<
		Record<string, boolean>
	>({
		ROLES: false,
		DEFAULT_ROLES: false,
		PROTECTED_ROLES: false,
		ALLOWED_ORIGINS: false,
		roles: false,
	});
	const [inputData, setInputData] = useState<Record<string, string>>({
		ROLES: '',
		DEFAULT_ROLES: '',
		PROTECTED_ROLES: '',
		ALLOWED_ORIGINS: '',
		roles: '',
	});
	const updateInputHandler = (
		type: string,
		operation: any,
		role: string = '',
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
				(item: string) => item !== role,
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
						},
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
					value={variables[inputType] || ''}
					onChange={(
						event: Event & {
							target: HTMLInputElement;
						},
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
				borderRadius={5}
				paddingTop="0.5%"
				overflowX={variables[inputType].length > 3 ? 'scroll' : 'hidden'}
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
										role,
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
							value={inputData[inputType] || ''}
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
	if (Object.values(MultiSelectInputType).includes(inputType)) {
		return (
			<Flex w="100%" style={{ position: 'relative' }}>
				<Flex
					border="1px solid #e2e8f0"
					w="100%"
					borderRadius="var(--chakra-radii-sm)"
					p="1% 0 0 2.5%"
					overflowX={variables[inputType].length > 3 ? 'scroll' : 'hidden'}
					overflowY="hidden"
					justifyContent="space-between"
					alignItems="center"
				>
					<Flex justifyContent="start" alignItems="center" w="100%" wrap="wrap">
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
												role,
											)
										}
									/>
								</Tag>
							</Box>
						))}
					</Flex>
					<Menu matchWidth={true}>
						<MenuButton px="10px" py="7.5px">
							<FaAngleDown />
						</MenuButton>
						<MenuList
							position="absolute"
							top="0"
							right="0"
							zIndex="10"
							maxH="150"
							overflowX="scroll"
						>
							<MenuOptionGroup
								title={undefined}
								value={variables[inputType]}
								type="checkbox"
								onChange={(values: string[] | string) => {
									setVariables({
										...variables,
										[inputType]: values,
									});
								}}
							>
								{availableUserRoles.map((role) => {
									return (
										<MenuItemOption
											key={`multiselect-menu-${role}`}
											value={role}
										>
											{role}
										</MenuItemOption>
									);
								})}
							</MenuOptionGroup>
						</MenuList>
					</Menu>
				</Flex>
			</Flex>
		);
	}
	if (Object.values(TextAreaInputType).includes(inputType)) {
		return (
			<Textarea
				{...props}
				size="lg"
				fontSize={14}
				value={variables[inputType] ? variables[inputType] : ''}
				onChange={(
					event: Event & {
						target: HTMLInputElement;
					},
				) =>
					setVariables({
						...variables,
						[inputType]: event.target.value,
					})
				}
			/>
		);
	}
	if (Object.values(SwitchInputType).includes(inputType)) {
		return (
			<Flex w="25%" justifyContent="space-between">
				<Text h="75%" fontWeight="bold" marginRight="2">
					Off
				</Text>
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
				<Text h="75%" fontWeight="bold" marginLeft="2">
					On
				</Text>
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
