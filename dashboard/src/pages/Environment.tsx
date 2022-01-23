import React from 'react';
import { Box, Divider, Flex, Stack, Center, Text } from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaGoogle, FaGithub, FaFacebookF } from 'react-icons/fa';
import InputField from '../components/InputField';
import { EnvVariablesQuery } from '../graphql/queries';
import { ArrayInputType, HiddenInputType, TextInputType } from '../constants';

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
}

export default function Environment() {
	const client = useClient();
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

	const updateEnvVariables = async () => {
		const {
			data: { _env: envData },
		} = await client.query(EnvVariablesQuery).toPromise();
		if (envData) {
			setEnvVariables({
				...envVariables,
				...envData,
				// test data
				GOOGLE_CLIENT_SECRET: 'xygchxcfcghjsvhccxgvgvxcz',
				GITHUB_CLIENT_SECRET: 'abvgxdgjbsgcxcjvjxvhcgfcxc',
				FACEBOOK_CLIENT_SECRET: 'pvhchbjhxvjhnklnhjvfcxqrh',
				GOOGLE_CLIENT_ID: 'jvhgvxcknbhjvc',
				GITHUB_CLIENT_ID: 'kxhvghchcxhjx',
				FACEBOOK_CLIENT_ID: 'gxgjbvxcgfcvghx',
			});
		}
	};

	React.useEffect(() => {
		updateEnvVariables();
	}, []);

	return (
		<Box m="5" p="5" bg="white" rounded="md">
			<Stack spacing={6}>
				<Text fontSize="md">Social Media Logins</Text>
				<Flex>
					<Center
						w="50px"
						margin="0 1.5% 0 5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaGoogle style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginRight="1.5%">
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
						/>
					</Center>
				</Flex>
				<Flex>
					<Center
						w="50px"
						margin="0 1.5% 0 5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaGithub style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginRight="1.5%">
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
						/>
					</Center>
				</Flex>
				<Flex>
					<Center
						w="50px"
						margin="0 1.5% 0 5%"
						border="1px solid #e2e8f0"
						borderRadius="5px"
					>
						<FaFacebookF style={{ color: '#8c8c8c' }} />
					</Center>
					<Center w="45%" marginRight="1.5%">
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
						/>
					</Center>
				</Flex>
				<Divider paddingTop="2%" />
			</Stack>
			<Stack spacing={6} paddingTop="3%">
				<Text fontSize="md">Roles</Text>
				<Flex>
					<Flex
						w="30%"
						marginLeft="5%"
						justifyContent="start"
						alignItems="center"
					>
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
					<Flex
						w="30%"
						marginLeft="5%"
						justifyContent="start"
						alignItems="center"
					>
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
					<Flex
						w="30%"
						marginLeft="5%"
						justifyContent="start"
						alignItems="center"
					>
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
				<Divider paddingTop="2%" />
			</Stack>
		</Box>
	);
}
