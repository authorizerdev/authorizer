import { Box, Center, Flex, Image, Text } from '@chakra-ui/react';
import React from 'react';
import { LOGO_URL } from '../constants';

export function AuthLayout({ children }: { children: React.ReactNode }) {
	return (
		<Flex
			flexWrap="wrap"
			h="100%"
			bg="gray.100"
			alignItems="center"
			justifyContent="center"
			flexDirection="column"
		>
			<Flex alignItems="center">
				<Image
					src="https://authorizer.dev/images/logo.png"
					alt="logo"
					height="50"
				/>
				<Text fontSize="x-large" ml="3" letterSpacing="3">
					AUTHORIZER
				</Text>
			</Flex>

			<Box p="6" m="5" rounded="5" bg="white" w="500px" shadow="xl">
				{children}
			</Box>
		</Flex>
	);
}
