import React from 'react';
import InputField from '../InputField';
import {
	Flex,
	Stack,
	Center,
	Text,
	Box,
	Divider,
	useMediaQuery,
} from '@chakra-ui/react';
import {
	FaGoogle,
	FaGithub,
	FaFacebookF,
	FaLinkedin,
	FaApple,
} from 'react-icons/fa';
import { TextInputType, HiddenInputType } from '../../constants';

const OAuthConfig = ({
	envVariables,
	setVariables,
	fieldVisibility,
	setFieldVisibility,
}: any) => {
	const [isNotSmallerScreen] = useMediaQuery('(min-width:667px)');
	return (
		<div>
			<Box>
				<Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={6}>
					Authorizer Config
				</Text>
				<Stack spacing={6} padding="2% 0%">
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">Client ID</Text>
						</Flex>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								variables={envVariables}
								setVariables={() => {}}
								inputType={TextInputType.CLIENT_ID}
								placeholder="Client ID"
								readOnly={true}
							/>
						</Center>
					</Flex>
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">Client Secret</Text>
						</Flex>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								variables={envVariables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.CLIENT_SECRET}
								placeholder="Client Secret"
								readOnly={true}
							/>
						</Center>
					</Flex>
				</Stack>
				<Divider mt={5} mb={2} color="blackAlpha.700" />
				<Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={4}>
					Social Media Logins
				</Text>
				<Stack spacing={6} padding="2% 0%">
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Center
							w={isNotSmallerScreen ? '55px' : '35px'}
							h="35px"
							marginRight="1.5%"
							border="1px solid #ff3e30"
							borderRadius="5px"
						>
							<FaGoogle style={{ color: '#ff3e30' }} />
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
							marginRight="1.5%"
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								inputType={TextInputType.GOOGLE_CLIENT_ID}
								placeholder="Google Client ID"
							/>
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.GOOGLE_CLIENT_SECRET}
								placeholder="Google Secret"
							/>
						</Center>
					</Flex>
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Center
							w={isNotSmallerScreen ? '55px' : '35px'}
							h="35px"
							marginRight="1.5%"
							border="1px solid #171515"
							borderRadius="5px"
						>
							<FaGithub style={{ color: '#171515' }} />
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
							marginRight="1.5%"
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								inputType={TextInputType.GITHUB_CLIENT_ID}
								placeholder="Github Client ID"
							/>
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.GITHUB_CLIENT_SECRET}
								placeholder="Github Secret"
							/>
						</Center>
					</Flex>
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Center
							w={isNotSmallerScreen ? '55px' : '35px'}
							h="35px"
							marginRight="1.5%"
							border="1px solid #3b5998"
							borderRadius="5px"
						>
							<FaFacebookF style={{ color: '#3b5998' }} />
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
							marginRight="1.5%"
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								inputType={TextInputType.FACEBOOK_CLIENT_ID}
								placeholder="Facebook Client ID"
							/>
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.FACEBOOK_CLIENT_SECRET}
								placeholder="Facebook Secret"
							/>
						</Center>
					</Flex>
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Center
							w={isNotSmallerScreen ? '55px' : '35px'}
							h="35px"
							marginRight="1.5%"
							border="1px solid #3b5998"
							borderRadius="5px"
						>
							<FaLinkedin style={{ color: '#3b5998' }} />
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
							marginRight="1.5%"
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								inputType={TextInputType.LINKEDIN_CLIENT_ID}
								placeholder="LinkedIn Client ID"
							/>
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.LINKEDIN_CLIENT_SECRET}
								placeholder="LinkedIn Client Secret"
							/>
						</Center>
					</Flex>
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Center
							w={isNotSmallerScreen ? '55px' : '35px'}
							h="35px"
							marginRight="1.5%"
							border="1px solid #3b5998"
							borderRadius="5px"
						>
							<FaApple style={{ color: '#3b5998' }} />
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
							marginRight="1.5%"
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								inputType={TextInputType.APPLE_CLIENT_ID}
								placeholder="Apple Client ID"
							/>
						</Center>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '3'}
						>
							<InputField
								borderRadius={5}
								variables={envVariables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.APPLE_CLIENT_SECRET}
								placeholder="Apple CLient Secret"
							/>
						</Center>
					</Flex>
				</Stack>
			</Box>
		</div>
	);
};

export default OAuthConfig;
