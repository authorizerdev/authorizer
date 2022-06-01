import React from 'react';
import { Flex, Stack, Center, Text, useMediaQuery } from '@chakra-ui/react';

import InputField from '../../components/InputField';
import { TextInputType } from '../../constants';

const DatabaseCredentials = ({ variables, setVariables }: any) => {
	const [isNotSmallerScreen] = useMediaQuery('(min-width:600px)');
	return (
		<div>
			{' '}
			<Text fontSize="md" paddingTop="2%" fontWeight="bold">
				Database Credentials
			</Text>
			<Stack spacing={6} padding="3% 0">
				<Text fontStyle="italic" fontSize="sm" color="blackAlpha.500" mt={3}>
					Note: Database related environment variables cannot be updated from
					dashboard. Please use .env file or OS environment variables to update
					it.
				</Text>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex
						w={isNotSmallerScreen ? '30%' : '40%'}
						justifyContent="start"
						alignItems="center"
					>
						<Text fontSize="sm">DataBase Name:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.DATABASE_NAME}
							isDisabled={true}
						/>
					</Center>
				</Flex>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex
						w={isNotSmallerScreen ? '30%' : '40%'}
						justifyContent="start"
						alignItems="center"
					>
						<Text fontSize="sm">DataBase Type:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.DATABASE_TYPE}
							isDisabled={true}
						/>
					</Center>
				</Flex>
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex
						w={isNotSmallerScreen ? '30%' : '40%'}
						justifyContent="start"
						alignItems="center"
					>
						<Text fontSize="sm">DataBase URL:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.DATABASE_URL}
							isDisabled={true}
						/>
					</Center>
				</Flex>
			</Stack>
		</div>
	);
};

export default DatabaseCredentials;
