import React from 'react';
import { Flex, Stack, Center, Text, useMediaQuery } from '@chakra-ui/react';
import InputField from '../../components/InputField';
import { TextInputType, HiddenInputType } from '../../constants';
const EmailConfigurations = ({
	variables,
	setVariables,
	fieldVisibility,
	setFieldVisibility,
}: any) => {
	const [isNotSmallerScreen] = useMediaQuery('(min-width:600px)');
	return (
		<div>
			{' '}
			<Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
				Email Configurations
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Host:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.SMTP_HOST}
						/>
					</Center>
				</Flex>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">SMTP Port:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.SMTP_PORT}
						/>
					</Center>
				</Flex>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex
						w={isNotSmallerScreen ? '30%' : '40%'}
						justifyContent="start"
						alignItems="center"
					>
						<Text fontSize="sm">SMTP Username:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.SMTP_USERNAME}
						/>
					</Center>
				</Flex>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex
						w={isNotSmallerScreen ? '30%' : '40%'}
						justifyContent="start"
						alignItems="center"
					>
						<Text fontSize="sm">SMTP Password:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							fieldVisibility={fieldVisibility}
							setFieldVisibility={setFieldVisibility}
							inputType={HiddenInputType.SMTP_PASSWORD}
						/>
					</Center>
				</Flex>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">From Email:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.SENDER_EMAIL}
						/>
					</Center>
				</Flex>
			</Stack>
		</div>
	);
};

export default EmailConfigurations;
