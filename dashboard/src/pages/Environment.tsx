import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Box, Flex, Stack, Button, useToast } from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaSave } from 'react-icons/fa';
import _ from 'lodash';
import { EnvVariablesQuery } from '../graphql/queries';
import {
	SelectInputType,
	HiddenInputType,
	TextInputType,
	HMACEncryptionType,
	RSAEncryptionType,
	ECDSAEncryptionType,
	envVarTypes,
	envSubViews,
} from '../constants';
import { UpdateEnvVariables } from '../graphql/mutation';
import { getObjectDiff, capitalizeFirstLetter } from '../utils';
import OAuthConfig from '../components/EnvComponents/OAuthConfig';
import Roles from '../components/EnvComponents/Roles';
import JWTConfigurations from '../components/EnvComponents/JWTConfiguration';
import SessionStorage from '../components/EnvComponents/SessionStorage';
import EmailConfigurations from '../components/EnvComponents/EmailConfiguration';
import DomainWhiteListing from '../components/EnvComponents/DomainWhitelisting';
import OrganizationInfo from '../components/EnvComponents/OrganizationInfo';
import AccessToken from '../components/EnvComponents/AccessToken';
import Features from '../components/EnvComponents/Features';
import SecurityAdminSecret from '../components/EnvComponents/SecurityAdminSecret';
import DatabaseCredentials from '../components/EnvComponents/DatabaseCredentials';

const Environment = () => {
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
		LINKEDIN_CLIENT_ID: '',
		LINKEDIN_CLIENT_SECRET: '',
		APPLE_CLIENT_ID: '',
		APPLE_CLIENT_SECRET: '',
		TWITTER_CLIENT_ID: '',
		TWITTER_CLIENT_SECRET: '',
		MICROSOFT_CLIENT_ID: '',
		MICROSOFT_CLIENT_SECRET: '',
		MICROSOFT_ACTIVE_DIRECTORY_TENANT_ID: '',
		TWITCH_CLIENT_ID: '',
		TWITCH_CLIENT_SECRET: '',
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
		SMTP_LOCAL_NAME: '',
		SENDER_EMAIL: '',
		SENDER_NAME: '',
		ALLOWED_ORIGINS: [],
		ORGANIZATION_NAME: '',
		ORGANIZATION_LOGO: '',
		CUSTOM_ACCESS_TOKEN_SCRIPT: '',
		ADMIN_SECRET: '',
		APP_COOKIE_SECURE: false,
		ADMIN_COOKIE_SECURE: false,
		DISABLE_LOGIN_PAGE: false,
		DISABLE_MAGIC_LINK_LOGIN: false,
		DISABLE_EMAIL_VERIFICATION: false,
		DISABLE_BASIC_AUTHENTICATION: false,
		DISABLE_SIGN_UP: false,
		DISABLE_STRONG_PASSWORD: false,
		OLD_ADMIN_SECRET: '',
		DATABASE_NAME: '',
		DATABASE_TYPE: '',
		DATABASE_URL: '',
		ACCESS_TOKEN_EXPIRY_TIME: '',
		DISABLE_MULTI_FACTOR_AUTHENTICATION: false,
		ENFORCE_MULTI_FACTOR_AUTHENTICATION: false,
		DEFAULT_AUTHORIZE_RESPONSE_TYPE: '',
		DEFAULT_AUTHORIZE_RESPONSE_MODE: '',
		DISABLE_PLAYGROUND: false,
		DISABLE_TOTP_LOGIN: false,
		DISABLE_MAIL_OTP_LOGIN: true,
	});

	const [fieldVisibility, setFieldVisibility] = React.useState<
		Record<string, boolean>
	>({
		GOOGLE_CLIENT_SECRET: false,
		GITHUB_CLIENT_SECRET: false,
		FACEBOOK_CLIENT_SECRET: false,
		LINKEDIN_CLIENT_SECRET: false,
		APPLE_CLIENT_SECRET: false,
		TWITTER_CLIENT_SECRET: false,
		TWITCH_CLIENT_SECRET: false,
		JWT_SECRET: false,
		SMTP_PASSWORD: false,
		ADMIN_SECRET: false,
		OLD_ADMIN_SECRET: false,
	});

	const { sec } = useParams();

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
	}, [sec]);

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
			{},
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
			position: 'top-right',
		});
	};

	const renderComponent = (tab: any) => {
		switch (tab) {
			case envSubViews.INSTANCE_INFO:
				return (
					<OAuthConfig
						envVariables={envVariables}
						setVariables={setEnvVariables}
						fieldVisibility={fieldVisibility}
						setFieldVisibility={setFieldVisibility}
					/>
				);
			case envSubViews.ROLES:
				return (
					<Roles variables={envVariables} setVariables={setEnvVariables} />
				);
			case envSubViews.JWT_CONFIG:
				return (
					<JWTConfigurations
						variables={envVariables}
						setVariables={setEnvVariables}
						fieldVisibility={fieldVisibility}
						setFieldVisibility={setFieldVisibility}
						SelectInputType={SelectInputType.JWT_TYPE}
						HMACEncryptionType={HMACEncryptionType}
						RSAEncryptionType={RSAEncryptionType}
						ECDSAEncryptionType={ECDSAEncryptionType}
						getData={getData}
					/>
				);
			case envSubViews.SESSION_STORAGE:
				return (
					<SessionStorage
						variables={envVariables}
						setVariables={setEnvVariables}
						RedisURL={TextInputType.REDIS_URL}
					/>
				);
			case envSubViews.EMAIL_CONFIG:
				return (
					<EmailConfigurations
						variables={envVariables}
						setVariables={setEnvVariables}
						fieldVisibility={fieldVisibility}
						setFieldVisibility={setFieldVisibility}
					/>
				);
			case envSubViews.WHITELIST_VARIABLES:
				return (
					<DomainWhiteListing
						variables={envVariables}
						setVariables={setEnvVariables}
					/>
				);
			case envSubViews.ORGANIZATION_INFO:
				return (
					<OrganizationInfo
						variables={envVariables}
						setVariables={setEnvVariables}
					/>
				);
			case envSubViews.ACCESS_TOKEN:
				return (
					<AccessToken
						variables={envVariables}
						setVariables={setEnvVariables}
					/>
				);
			case envSubViews.FEATURES:
				return (
					<Features variables={envVariables} setVariables={setEnvVariables} />
				);
			case envSubViews.ADMIN_SECRET:
				return (
					<SecurityAdminSecret
						variables={envVariables}
						setVariables={setEnvVariables}
						fieldVisibility={fieldVisibility}
						setFieldVisibility={setFieldVisibility}
						validateAdminSecretHandler={validateAdminSecretHandler}
						adminSecret={adminSecret}
					/>
				);
			case envSubViews.DB_CRED:
				return (
					<DatabaseCredentials
						variables={envVariables}
						setVariables={setEnvVariables}
					/>
				);
			default:
				return (
					<OAuthConfig
						envVariables={envVariables}
						setVariables={setEnvVariables}
						fieldVisibility={fieldVisibility}
						setFieldVisibility={setFieldVisibility}
					/>
				);
		}
	};
	return (
		<Box m="5" py="5" px="10" bg="white" rounded="md">
			{renderComponent(sec)}
			<Stack spacing={6} padding="1% 0" mt={4}>
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
};

export default Environment;
