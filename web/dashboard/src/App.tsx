import * as React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { createClient, Provider } from 'urql';
import { cacheExchange, fetchExchange } from 'urql';
import { Toaster } from 'sonner';
import { TooltipProvider } from './components/ui/tooltip';
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
	exchanges: [cacheExchange, fetchExchange],
});

export default function App() {
	return (
		<TooltipProvider>
			<Provider value={queryClient}>
				<BrowserRouter basename="/dashboard">
					<AuthContextProvider>
						<AppRoutes />
					</AuthContextProvider>
				</BrowserRouter>
			</Provider>
			<Toaster position="top-right" richColors closeButton />
		</TooltipProvider>
	);
}
