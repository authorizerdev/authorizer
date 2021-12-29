import * as React from 'react';
import { Text, ChakraProvider } from '@chakra-ui/react';
import { MdStar } from 'react-icons/md';

export default function Example() {
	return (
		<ChakraProvider>
			<Text
				ml={2}
				textTransform="uppercase"
				fontSize="xl"
				fontWeight="bold"
				color="pink.800"
			>
				Authorizer Dashboard
			</Text>
		</ChakraProvider>
	);
}
