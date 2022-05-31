import React from 'react';
import { Flex, Stack, Center, Text, useMediaQuery } from '@chakra-ui/react';
import InputField from '../InputField';

const SessionStorage = ({ variables, setVariables, RedisURL }: any) => {
	const [isNotSmallerScreen] = useMediaQuery('(min-width:600px)');
	return (
		<div>
			{' '}
			<Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
				Session Storage
			</Text>
			<Text fontStyle="italic" fontSize="sm" color="blackAlpha.500" mt={3}>
				Note: Redis related environment variables cannot be updated from
				dashboard. Please use .env file or OS environment variables to update
				it.
			</Text>
			<Stack spacing={6} padding="2% 0%">
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">Redis URL:</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '3'}
					>
						<InputField
							disabled
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={RedisURL}
							placeholder="Redis URL"
						/>
					</Center>
				</Flex>
			</Stack>
		</div>
	);
};

export default SessionStorage;
