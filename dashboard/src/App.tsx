import * as React from 'react';
import { Text, ChakraProvider } from '@chakra-ui/react';
import { MdStar } from 'react-icons/md';
import { BrowserRouter } from 'react-router-dom';

export default function Example() {
	return (
		<ChakraProvider>
			<BrowserRouter>
				<h1>Dashboard</h1>
			</BrowserRouter>
		</ChakraProvider>
	);
}
