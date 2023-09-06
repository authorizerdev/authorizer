import React from 'react';
import { Divider, Flex, Stack, Text } from '@chakra-ui/react';
import InputField from '../InputField';
import { SwitchInputType } from '../../constants';

const Features = ({ variables, setVariables }: any) => {
	// window.alert(variables)
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
							hasReversedValue
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
							hasReversedValue
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
							hasReversedValue
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
							hasReversedValue
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
							hasReversedValue
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
							hasReversedValue
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
							hasReversedValue
						/>
					</Flex>
				</Flex>

				{
					!variables.DISABLE_MULTI_FACTOR_AUTHENTICATION &&
					<Flex alignItems="center">
						<Flex w="100%" alignItems="baseline" flexDir="column">
							<Text fontSize="sm">TOTP:</Text>
							<Text fontSize="x-small">
								Note: to enable totp mfa
							</Text>
						</Flex>

						<Flex justifyContent="start" mb={3}>
							<InputField
								variables={variables}
								setVariables={setVariables}
								inputType={SwitchInputType.DISABLE_TOTP_LOGIN}
								hasReversedValue
							/>
						</Flex>
					</Flex>
				}
				{!variables.DISABLE_MULTI_FACTOR_AUTHENTICATION &&
					<Flex alignItems="center">
					<Flex w="100%" alignItems="baseline" flexDir="column">
					<Text fontSize="sm">EMAIL OTP:</Text>
					<Text fontSize="x-small">
					Note: to enable email otp mfa
					</Text>
					</Flex>

					<Flex justifyContent="start" mb={3}>
				<InputField
					variables={variables}
					setVariables={setVariables}
					inputType={SwitchInputType.DISABLE_MAIL_OTP_LOGIN}
					hasReversedValue
				/>
			</Flex>
		</Flex>}

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
						/>
					</Flex>
				</Flex>
				<Flex>
					<Flex w="100%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Playground:</Text>
					</Flex>
					<Flex justifyContent="start">
						<InputField
							variables={variables}
							setVariables={setVariables}
							inputType={SwitchInputType.DISABLE_PLAYGROUND}
							hasReversedValue
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
