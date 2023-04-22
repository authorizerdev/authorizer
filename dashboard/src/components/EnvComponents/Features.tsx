import React from 'react';
import { Divider, Flex, Stack, Text } from '@chakra-ui/react';
import InputField from '../InputField';
import { SwitchInputType } from '../../constants';

const Features = ({ variables, setVariables }: any) => {
	return (
		<div>
			{' '}
			<Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
				Features
			</Text>
			<Stack spacing={6}>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Login Page:</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_LOGIN_PAGE}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Email Verification:</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_EMAIL_VERIFICATION}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Magic Login Link:</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_MAGIC_LINK_LOGIN}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Basic Authentication:</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_BASIC_AUTHENTICATION}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Sign Up:</Text>
					</Flex>
					<Flex justifyContent="start" mb={3}>
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_SIGN_UP}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Strong Password:</Text>
					</Flex>
					<Flex justifyContent="start" mb={3}>
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_STRONG_PASSWORD}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex alignItems="center">
					<Flex w="100%" alignItems="baseline" flexDir="column">
						<Text fontSize="sm">Multi Factor Authentication (MFA):</Text>
						<Text fontSize="x-small">
							Note: Enabling this will ignore Enforcing MFA shown below and will
							also ignore the user MFA setting.
						</Text>
					</Flex>
					<Flex justifyContent="start" mb={3}>
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_MULTI_FACTOR_AUTHENTICATION}
							is_Disable={true}
						/>
					</Flex>
				</Flex>
				<Flex alignItems="center">
					<Flex w="100%" alignItems="baseline" flexDir="column">
						<Text fontSize="sm">
							Enforce Multi Factor Authentication (MFA):
						</Text>
						<Text fontSize="x-small">
							Note: If you disable enforcing after it was enabled, it will still
							keep MFA enabled for older users.
						</Text>
					</Flex>
					<Flex justifyContent="start" mb={3}>
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.ENFORCE_MULTI_FACTOR_AUTHENTICATION}
							is_Disable={false}
						/>
					</Flex>
				</Flex>
			</Stack>
			<Divider paddingY={5} />
			<Text fontSize="md" paddingTop={5} fontWeight="bold" mb={5}>
				Cookie Security Features
			</Text>
			<Stack spacing={6}>
				<Flex>
					<Flex w="100%" alignItems="baseline" flexDir="column">
						<Text fontSize="sm">Use Secure App Cookie:</Text>
						<Text fontSize="x-small">
							Note: If you set this to insecure, it will set{' '}
							<code>sameSite</code> property of cookie to <code>lax</code> mode
						</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.APP_COOKIE_SECURE}
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" alignItems="baseline" flexDir="column">
						<Text fontSize="sm">Use Secure Admin Cookie:</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.ADMIN_COOKIE_SECURE}
						/>
					</Flex>
				</Flex>
			</Stack>
		</div>
	);
};

export default Features;
