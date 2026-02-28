import { Text } from '@chakra-ui/react';
import React from 'react';

export default function Home() {
	return (
		<>
			<Text fontSize="2xl" fontWeight="bold">
				Hi there 👋 <br />
			</Text>

			<Text fontSize="xl" color="gray.700">
				Welcome to Authorizer Administrative Dashboard! <br />
				Manage your users, webhooks, and email templates.
			</Text>
		</>
	);
}
