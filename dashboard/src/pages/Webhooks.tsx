import React from 'react';
import { Box, Flex, Text } from '@chakra-ui/react';
import AddWebhookModal from '../components/AddWebhookModal';

const Webhooks = () => {
	return (
		<Box m="5" py="5" px="10" bg="white" rounded="md">
			<Flex margin="2% 0" justifyContent="space-between" alignItems="center">
				<Text fontSize="md" fontWeight="bold">
					Webhooks
				</Text>
				<AddWebhookModal />
			</Flex>
		</Box>
	);
};

export default Webhooks;
