import { Box, Flex } from '@chakra-ui/react';
import React from 'react';
import { Sidebar } from '../components/Sidebar';

export function DashboardLayout({ children }: { children: React.ReactNode }) {
	return (
		<Flex flexWrap="wrap" h="100%">
			<Box w="72" bg="blue.500" flex="1" position="fixed" h="100vh">
				<Sidebar />
			</Box>
			<Box as="main" flex="2" p="10" marginLeft="72">
				{children}
			</Box>
		</Flex>
	);
}
