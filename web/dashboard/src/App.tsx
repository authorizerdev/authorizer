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
	// The server serves GraphQL on POST only (internal/server/http_routes.go).
	// @urql/core 6 sends queries as GET by default ("within-url-limit"), which
	// 404s every query: no meta, no admin session, an empty dashboard.
	preferGetMethod: false,
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
