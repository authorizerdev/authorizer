import React, { useEffect } from 'react';
import {
	Box,
	Divider,
	Flex,
	Stack,
	Center,
	Text,
	Button,
	Input,
	InputGroup,
	InputRightElement,
	useToast,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import {
	FaGoogle,
	FaGithub,
	FaFacebookF,
	FaSave,
	FaRegEyeSlash,
	FaRegEye,
} from 'react-icons/fa';
import _ from 'lodash';
import InputField from '../components/InputField';
import { EnvVariablesQuery } from '../graphql/queries';
import {
	ArrayInputType,
	SelectInputType,
	HiddenInputType,
	TextInputType,
	TextAreaInputType,
	SwitchInputType,
	HMACEncryptionType,
	RSAEncryptionType,
	ECDSAEncryptionType,
	envVarTypes,
} from '../constants';
import { UpdateEnvVariables } from '../graphql/mutation';
import { getObjectDiff, capitalizeFirstLetter } from '../utils';
import GenerateKeysModal from '../components/GenerateKeysModal';

export default function Environment() {
	const client = useClient();
	const toast = useToast();
	const [adminSecret, setAdminSecret] = React.useState<
		Record<string, string | boolean>
	>({
		value: '',
		disableInputField: true,
	});
	const [loading, setLoading] = React.useState<boolean>(true);
	const [envVariables, setEnvVariables] = React.useState<envVarTypes>({
		GOOGLE_CLIENT_ID: '',
		GOOGLE_CLIENT_SECRET: '',
		GITHUB_CLIENT_ID: '',
		GITHUB_CLIENT_SECRET: '',
		FACEBOOK_CLIENT_ID: '',
		FACEBOOK_CLIENT_SECRET: '',
		ROLES: [],
		DEFAULT_ROLES: [],
		PROTECTED_ROLES: [],
		JWT_TYPE: '',
		JWT_SECRET: '',
		JWT_ROLE_CLAIM: '',
		JWT_PRIVATE_KEY: '',
		JWT_PUBLIC_KEY: '',
		REDIS_URL: '',
		SMTP_HOST: '',
		SMTP_PORT: '',
		SMTP_USERNAME: '',
		SMTP_PASSWORD: '',
		SENDER_EMAIL: '',
		ALLOWED_ORIGINS: [],
		ORGANIZATION_NAME: '',
		ORGANIZATION_LOGO: '',
		CUSTOM_ACCESS_TOKEN_SCRIPT: '',
		ADMIN_SECRET: '',
		DISABLE_LOGIN_PAGE: false,
		DISABLE_MAGIC_LINK_LOGIN: false,
		DISABLE_EMAIL_VERIFICATION: false,
		DISABLE_BASIC_AUTHENTICATION: false,
		DISABLE_SIGN_UP: false,
		OLD_ADMIN_SECRET: '',
		DATABASE_NAME: '',
		DATABASE_TYPE: '',
		DATABASE_URL: '',
	});

	const [fieldVisibility, setFieldVisibility] = React.useState<
		Record<string, boolean>
	>({
		GOOGLE_CLIENT_SECRET: false,
		GITHUB_CLIENT_SECRET: false,
		FACEBOOK_CLIENT_SECRET: false,
		JWT_SECRET: false,
		SMTP_PASSWORD: false,
		ADMIN_SECRET: false,
		OLD_ADMIN_SECRET: false,
	});

	async function getData() {
		const {
			data: { _env: envData },
		} = await client.query(EnvVariablesQuery).toPromise();
		setLoading(false);
		setEnvVariables({
			...envData,
			OLD_ADMIN_SECRET: envData.ADMIN_SECRET,
			ADMIN_SECRET: '',
		});
		setAdminSecret({
			value: '',
			disableInputField: true,
		});
	}

	useEffect(() => {
		getData();
	}, []);

	const validateAdminSecretHandler = (event: any) => {
		if (envVariables.OLD_ADMIN_SECRET === event.target.value) {
			setAdminSecret({
				...adminSecret,
				value: event.target.value,
				disableInputField: false,
			});
		} else {
			setAdminSecret({
				...adminSecret,
				value: event.target.value,
				disableInputField: true,
			});
		}
		if (envVariables.ADMIN_SECRET !== '') {
			setEnvVariables({ ...envVariables, ADMIN_SECRET: '' });
		}
	};

	const saveHandler = async () => {
		setLoading(true);
		const {
			data: { _env: envData },
		} = await client.query(EnvVariablesQuery).toPromise();
		const diff = getObjectDiff(envVariables, envData);
		const updatedEnvVariables = diff.reduce(
			(acc: any, property: string) => ({
				...acc,
				// @ts-ignore
				[property]: envVariables[property],
			}),
			{}
		);
		if (
			updatedEnvVariables[HiddenInputType.ADMIN_SECRET] === '' ||
			updatedEnvVariables[HiddenInputType.OLD_ADMIN_SECRET] !==
				envData.ADMIN_SECRET
		) {
			delete updatedEnvVariables.OLD_ADMIN_SECRET;
			delete updatedEnvVariables.ADMIN_SECRET;
		}

		delete updatedEnvVariables.DATABASE_URL;
		delete updatedEnvVariables.DATABASE_TYPE;
		delete updatedEnvVariables.DATABASE_NAME;

		const res = await client
			.mutation(UpdateEnvVariables, { params: updatedEnvVariables })
			.toPromise();

		setLoading(false);

		if (res.error) {
			toast({
				title: capitalizeFirstLetter(res.error.message),
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});

			return;
		}

		setAdminSecret({
			value: '',
			disableInputField: true,
		});

		getData();

		toast({
			title: `Successfully updated ${
				Object.keys(updatedEnvVariables).length
			} variables`,
			isClosable: true,
			status: 'success',
			position: 'bottom-right',
		});
	};

	return (
		<Box m="5" py="5" px="10" bg="white" rounded="md">
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Your instance information
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Client ID</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={() => {}}
							inputType={TextInputType.CLIENT_ID}
							placeholder="Client ID"
							readOnly={true}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Client Secret</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.CLIENT_SECRET}
							placeholder="Client Secret"
							readOnly={true}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Social Media Logins
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Center
						w="50px"
						marginRight="1.5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaGoogle style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginRight="1.5%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.GOOGLE_CLIENT_ID}
							placeholder="Google Client ID"
						/>
					</Center>
					<Center w="45%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.GOOGLE_CLIENT_SECRET}
							placeholder="Google Secret"
						/>
					</Center>
				</Flex>
				<Flex>
					<Center
						w="50px"
						marginRight="1.5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaGithub style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginRight="1.5%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.GITHUB_CLIENT_ID}
							placeholder="Github Client ID"
						/>
					</Center>
					<Center w="45%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.GITHUB_CLIENT_SECRET}
							placeholder="Github Secret"
						/>
					</Center>
				</Flex>
				<Flex>
					<Center
						w="50px"
						marginRight="1.5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaFacebookF style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginRight="1.5%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.FACEBOOK_CLIENT_ID}
							placeholder="Facebook Client ID"
						/>
					</Center>
					<Center w="45%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.FACEBOOK_CLIENT_SECRET}
							placeholder="Facebook Secret"
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Roles
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Roles:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={ArrayInputType.ROLES}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Default Roles:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={ArrayInputType.DEFAULT_ROLES}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Protected Roles:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={ArrayInputType.PROTECTED_ROLES}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Flex
				width="100%"
				justifyContent="space-between"
				alignItems="center"
				paddingTop="2%"
			>
				<Text fontSize="md" fontWeight="bold">
					JWT (JSON Web Tokens) Configurations
				</Text>
				<Flex>
					<GenerateKeysModal
						jwtType={envVariables.JWT_TYPE}
						getData={getData}
					/>
				</Flex>
			</Flex>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">JWT Type:</Text>
					</Flex>
					<Flex w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={SelectInputType.JWT_TYPE}
							value={SelectInputType.JWT_TYPE}
							options={{
								...HMACEncryptionType,
								...RSAEncryptionType,
								...ECDSAEncryptionType,
							}}
						/>
					</Flex>
				</Flex>
				{Object.values(HMACEncryptionType).includes(envVariables.JWT_TYPE) ? (
					<Flex>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">JWT Secret</Text>
						</Flex>
						<Center w="70%">
							<InputField
								variables={envVariables}
								setVariables={setEnvVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.JWT_SECRET}
							/>
						</Center>
					</Flex>
				) : (
					<>
						<Flex>
							<Flex w="30%" justifyContent="start" alignItems="center">
								<Text fontSize="sm">Public Key</Text>
							</Flex>
							<Center w="70%">
								<InputField
									variables={envVariables}
									setVariables={setEnvVariables}
									inputType={TextAreaInputType.JWT_PUBLIC_KEY}
									placeholder="Add public key here"
									minH="25vh"
								/>
							</Center>
						</Flex>
						<Flex>
							<Flex w="30%" justifyContent="start" alignItems="center">
								<Text fontSize="sm">Private Key</Text>
							</Flex>
							<Center w="70%">
								<InputField
									variables={envVariables}
									setVariables={setEnvVariables}
									inputType={TextAreaInputType.JWT_PRIVATE_KEY}
									placeholder="Add private key here"
									minH="25vh"
								/>
							</Center>
						</Flex>
					</>
				)}
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">JWT Role Claim:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.JWT_ROLE_CLAIM}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Session Storage
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Redis URL:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.REDIS_URL}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Email Configurations
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Host:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.SMTP_HOST}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Port:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.SMTP_PORT}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Username:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.SMTP_USERNAME}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Password:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.SMTP_PASSWORD}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">From Email:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.SENDER_EMAIL}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				White Listing
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Allowed Origins:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={ArrayInputType.ALLOWED_ORIGINS}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Organization Information
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Organization Name:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.ORGANIZATION_NAME}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Organization Logo:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextInputType.ORGANIZATION_LOGO}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Custom Access Token Scripts
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Center w="100%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={TextAreaInputType.CUSTOM_ACCESS_TOKEN_SCRIPT}
							placeholder="Add script here"
							minH="25vh"
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Disable Features
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Disable Login Page:</Text>
					</Flex>
					<Flex justifyContent="start" w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_LOGIN_PAGE}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Disable Email Verification:</Text>
					</Flex>
					<Flex justifyContent="start" w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_EMAIL_VERIFICATION}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Disable Magic Login Link:</Text>
					</Flex>
					<Flex justifyContent="start" w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_MAGIC_LINK_LOGIN}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Disable Basic Authentication:</Text>
					</Flex>
					<Flex justifyContent="start" w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_BASIC_AUTHENTICATION}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Disable Sign Up:</Text>
					</Flex>
					<Flex justifyContent="start" w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_SIGN_UP}
						/>
					</Flex>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Danger
			</Text>
			<Stack
				spacing={6}
				padding="0 5%"
				marginTop="3%"
				border="1px solid #ff7875"
				borderRadius="5px"
			>
				<Stack spacing={6} padding="3% 0">
					<Text fontStyle="italic" fontSize="sm" color="gray.600">
						Note: Database related environment variables cannot be updated from
						dashboard :(
					</Text>
					<Flex>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">DataBase Name:</Text>
						</Flex>
						<Center w="70%">
							<InputField
								variables={envVariables}
								setVariables={setEnvVariables}
								inputType={TextInputType.DATABASE_NAME}
								isDisabled={true}
							/>
						</Center>
					</Flex>
					<Flex>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">DataBase Type:</Text>
						</Flex>
						<Center w="70%">
							<InputField
								variables={envVariables}
								setVariables={setEnvVariables}
								inputType={TextInputType.DATABASE_TYPE}
								isDisabled={true}
							/>
						</Center>
					</Flex>
					<Flex>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">DataBase URL:</Text>
						</Flex>
						<Center w="70%">
							<InputField
								variables={envVariables}
								setVariables={setEnvVariables}
								inputType={TextInputType.DATABASE_URL}
								isDisabled={true}
							/>
						</Center>
					</Flex>
				</Stack>
				<Flex marginTop="3%">
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Old Admin Secret:</Text>
					</Flex>
					<Center w="70%">
						<InputGroup size="sm">
							<Input
								size="sm"
								placeholder="Enter Old Admin Secret"
								value={adminSecret.value as string}
								onChange={(event: any) => validateAdminSecretHandler(event)}
								type={
									!fieldVisibility[HiddenInputType.OLD_ADMIN_SECRET]
										? 'password'
										: 'text'
								}
							/>
							<InputRightElement
								right="5px"
								children={
									<Flex>
										{fieldVisibility[HiddenInputType.OLD_ADMIN_SECRET] ? (
											<Center
												w="25px"
												margin="0 1.5%"
												cursor="pointer"
												onClick={() =>
													setFieldVisibility({
														...fieldVisibility,
														[HiddenInputType.OLD_ADMIN_SECRET]: false,
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
														[HiddenInputType.OLD_ADMIN_SECRET]: true,
													})
												}
											>
												<FaRegEye color="#bfbfbf" />
											</Center>
										)}
									</Flex>
								}
							/>
						</InputGroup>
					</Center>
				</Flex>
				<Flex paddingBottom="3%">
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">New Admin Secret:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							variables={envVariables}
							setVariables={setEnvVariables}
							inputType={HiddenInputType.ADMIN_SECRET}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							isDisabled={adminSecret.disableInputField}
							placeholder="Enter New Admin Secret"
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="5%" marginBottom="2%" />
			<Stack spacing={6} padding="1% 0">
				<Flex justifyContent="end" alignItems="center">
					<Button
						leftIcon={<FaSave />}
						colorScheme="blue"
						variant="solid"
						onClick={saveHandler}
						isDisabled={loading}
					>
						Save
					</Button>
				</Flex>
			</Stack>
		</Box>
	);
}
