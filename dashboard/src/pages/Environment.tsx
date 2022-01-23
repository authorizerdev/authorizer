import React from 'react';
import {
	Box,
	Divider,
	Flex,
	Stack,
	Center,
	Text,
	Button,
	Input,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import {
	FaGoogle,
	FaGithub,
	FaFacebookF,
	FaUndo,
	FaSave,
} from 'react-icons/fa';
import InputField from '../components/InputField';
import { EnvVariablesQuery } from '../graphql/queries';
import {
	ArrayInputType,
	SelectInputType,
	HiddenInputType,
	TextInputType,
	TextAreaInputType,
	SwitchInputType,
} from '../constants';

interface envVarTypes {
	GOOGLE_CLIENT_ID: string;
	GOOGLE_CLIENT_SECRET: string;
	GITHUB_CLIENT_ID: string;
	GITHUB_CLIENT_SECRET: string;
	FACEBOOK_CLIENT_ID: string;
	FACEBOOK_CLIENT_SECRET: string;
	ROLES: [string] | [];
	DEFAULT_ROLES: [string] | [];
	PROTECTED_ROLES: [string] | [];
	JWT_TYPE: string;
	JWT_SECRET: string;
	JWT_ROLE_CLAIM: string;
	REDIS_URL: string;
	SMTP_HOST: string;
	SMTP_PORT: string;
	SMTP_USERNAME: string;
	SMTP_PASSWORD: string;
	SENDER_EMAIL: string;
	ALLOWED_ORIGINS: [string] | [];
	ORGANIZATION_NAME: string;
	ORGANIZATION_LOGO: string;
	CUSTOM_ACCESS_TOKEN_SCRIPT: string;
	ADMIN_SECRET: string;
	DISABLE_LOGIN_PAGE: boolean;
	DISABLE_MAGIC_LINK_LOGIN: boolean;
	DISABLE_EMAIL_VERIFICATION: boolean;
	DISABLE_BASIC_AUTHENTICATION: boolean;
}

export default function Environment() {
	const client = useClient();
	const [adminSecret, setAdminSecret] = React.useState<
		Record<string, string | boolean>
	>({
		oldValue: '',
		newValue: '',
		disableInputField: true,
	});
	const [loading, setLoading] = React.useState<boolean>(false);
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
	});

	const updateHandler = async () => {
		setLoading(true);
		const {
			data: { _env: envData },
		} = await client.query(EnvVariablesQuery).toPromise();
		console.log('envData ==>> ', envData);
		if (envData) {
			setAdminSecret({ ...adminSecret, oldValue: envData.ADMIN_SECRET });
			setEnvVariables({
				...envVariables,
				...envData,
				ADMIN_SECRET: '',
				// test data
				GOOGLE_CLIENT_SECRET: 'xygchxcfcghjsvhccxgvgvxcz',
				GITHUB_CLIENT_SECRET: 'abvgxdgjbsgcxcjvjxvhcgfcxc',
				FACEBOOK_CLIENT_SECRET: 'pvhchbjhxvjhnklnhjvfcxqrh',
				GOOGLE_CLIENT_ID: 'jvhgvxcknbhjvc',
				GITHUB_CLIENT_ID: 'kxhvghchcxhjx',
				FACEBOOK_CLIENT_ID: 'gxgjbvxcgfcvghx',
				REDIS_URL: 'redis://rediscloud:mypassword@redis...',
			});
		}
		setLoading(false);
	};

	React.useEffect(() => {
		updateHandler();
	}, []);

	const validateAdminSecretHandler = (event: any) => {
		if (adminSecret.oldValue === event.target.value) {
			setAdminSecret({
				...adminSecret,
				newValue: event.target.value,
				disableInputField: false,
			});
		} else {
			setAdminSecret({
				...adminSecret,
				newValue: event.target.value,
				disableInputField: true,
			});
		}
		if (envVariables.ADMIN_SECRET !== '') {
			setEnvVariables({ ...envVariables, ADMIN_SECRET: '' });
		}
	};

	const saveHandler = () => {
		console.log('updated env vars ==>> ', envVariables);
		updateHandler();
	};

	return (
		<Box m="5" p="5" bg="white" rounded="md">
			<Text fontSize="md" paddingTop="2%">
				Social Media Logins
			</Text>
			<Stack spacing={6} padding="3%">
				<Flex>
					<Center
						w="50px"
						marginLeft="1.5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaGoogle style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginLeft="1%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.GOOGLE_CLIENT_ID}
							placeholder="Google Client ID"
						/>
					</Center>
					<Center w="45%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.GOOGLE_CLIENT_SECRET}
							placeholder="Google Secret"
							marginLeft="2%"
						/>
					</Center>
				</Flex>
				<Flex>
					<Center
						w="50px"
						marginLeft="1.5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaGithub style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginLeft="1%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.GITHUB_CLIENT_ID}
							placeholder="Github Client ID"
						/>
					</Center>
					<Center w="45%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.GITHUB_CLIENT_SECRET}
							placeholder="Github Secret"
							marginLeft="2%"
						/>
					</Center>
				</Flex>
				<Flex>
					<Center
						w="50px"
						marginLeft="1.5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaFacebookF style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginLeft="1%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.FACEBOOK_CLIENT_ID}
							placeholder="Facebook Client ID"
						/>
					</Center>
					<Center w="45%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.FACEBOOK_CLIENT_SECRET}
							placeholder="Facebook Secret"
							marginLeft="2%"
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Roles
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Roles:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
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
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
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
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={ArrayInputType.PROTECTED_ROLES}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				JWT Configurations
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">JWT Type:</Text>
					</Flex>
					<Center w="70%">
						<Flex w="100%" justifyContent="space-between">
							<Flex flex="2">
								<InputField
									envVariables={envVariables}
									setEnvVariables={setEnvVariables}
									inputType={SelectInputType.JWT_TYPE}
									isDisabled={true}
									defaultValue={SelectInputType.JWT_TYPE}
								/>
							</Flex>
							<Flex flex="3" justifyContent="center" alignItems="center">
								<Text fontSize="sm">
									More JWT types will be enabled in upcoming releases.
								</Text>
							</Flex>
						</Flex>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">JWT Secret</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.JWT_SECRET}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">JWT Role Claim:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.JWT_ROLE_CLAIM}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Session Storage
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Redis URL:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.REDIS_URL}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Email Configurations
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Host:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.SMTP_HOST}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">PORT:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.SMTP_PORT}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Username:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.SMTP_USERNAME}
						/>
					</Center>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Password:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
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
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.SENDER_EMAIL}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				White Listing
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Allowed Origins:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={ArrayInputType.ALLOWED_ORIGINS}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Organization Information
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Organization Name:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
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
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextInputType.ORGANIZATION_LOGO}
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Custom Scripts
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Center w="100%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={TextAreaInputType.CUSTOM_ACCESS_TOKEN_SCRIPT}
							placeholder="Add script here"
							minH="25vh"
						/>
					</Center>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Disable Features
			</Text>
			<Stack spacing={6} padding="3% 5%">
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Login Page:</Text>
					</Flex>
					<Flex justifyContent="center" w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_LOGIN_PAGE}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Magic Login Link:</Text>
					</Flex>
					<Flex justifyContent="center" w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_MAGIC_LINK_LOGIN}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Email Verification:</Text>
					</Flex>
					<Flex justifyContent="center" w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_EMAIL_VERIFICATION}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Basic Authentication:</Text>
					</Flex>
					<Flex justifyContent="center" w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
							inputType={SwitchInputType.DISABLE_BASIC_AUTHENTICATION}
						/>
					</Flex>
				</Flex>
			</Stack>
			<Divider marginTop="2%" marginBottom="2%" />
			<Text fontSize="md" paddingTop="2%">
				Danger
			</Text>
			<Stack
				spacing={6}
				bgColor="#fff1f0"
				padding="0 5%"
				marginTop="3%"
				border="1px solid #ff7875"
				borderRadius="5px"
			>
				<Flex marginTop="3%">
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Old Admin Secret:</Text>
					</Flex>
					<Center w="70%">
						<Input
							size="sm"
							placeholder="Enter Old Admin Secret"
							value={adminSecret.newValue as string}
							onChange={(event: any) => validateAdminSecretHandler(event)}
						/>
					</Center>
				</Flex>
				<Flex paddingBottom="3%">
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">New Admin Secret:</Text>
					</Flex>
					<Center w="70%">
						<InputField
							envVariables={envVariables}
							setEnvVariables={setEnvVariables}
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
					<Stack direction="row" spacing={4}>
						<Button
							leftIcon={<FaUndo />}
							colorScheme="blue"
							variant="outline"
							onClick={updateHandler}
							isDisabled={loading}
						>
							Reset
						</Button>
						<Button
							leftIcon={<FaSave />}
							colorScheme="blue"
							variant="solid"
							onClick={saveHandler}
							isDisabled={loading}
						>
							Save
						</Button>
					</Stack>
				</Flex>
			</Stack>
		</Box>
	);
}
