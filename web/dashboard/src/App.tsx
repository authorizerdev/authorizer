import * as React from 'react';
import { Fragment } from 'react';
import { ChakraProvider, extendTheme } from '@chakra-ui/react';
import { BrowserRouter } from 'react-router-dom';
import { createClient, Provider } from 'urql';
import { AppRoutes } from './routes';
import { AuthContextProvider } from './contexts/AuthContext';

const queryClient = createClient({
	url: '/graphql',
	fetchOptions: () => {
		return {
			credentials: 'include',
			headers: {
				'x-authorizer-url': window.location.origin,
			},
		};
	},
	requestPolicy: 'network-only',
});

const theme = extendTheme({
	styles: {
		global: {
			'html, body, #root': {
				height: '100%',
				outline: 'none',
			},
		},
	},
	colors: {
		blue: {
			500: 'rgb(59,130,246)',
		},
	},
});

export default function App() {
	return (
		<Fragment>
			<ChakraProvider theme={theme}>
				<Provider value={queryClient}>
					<BrowserRouter basename="/dashboard">
						<AuthContextProvider>
							<AppRoutes />
						</AuthContextProvider>
					</BrowserRouter>
				</Provider>
			</ChakraProvider>
		</Fragment>
	);
}
