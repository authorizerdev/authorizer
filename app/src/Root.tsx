import React, { useEffect, lazy, Suspense } from 'react';
import { Switch, Route } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));

export default function Root({
	globalState,
}: {
	globalState: Record<string, string>;
}) {
	const { token, loading, config } = useAuthorizer();

	useEffect(() => {
		if (token) {
			let redirectURL = config.redirectURL || '/app';
			let params = `access_token=${token.access_token}&id_token=${token.id_token}&expires_in=${token.expires_in}&state=${globalState.state}`;
			if (token.refresh_token) {
				params += `&refresh_token=${token.refresh_token}`;
			}
			const url = new URL(redirectURL);
			if (redirectURL.includes('?')) {
				redirectURL = `${redirectURL}&${params}`;
			} else {
				redirectURL = `${redirectURL}?${params}`;
			}

			if (url.origin !== window.location.origin) {
				sessionStorage.removeItem('authorizer_state');
				window.location.replace(redirectURL);
			}
		}
		return () => {};
	}, [token]);

	if (loading) {
		return <h1>Loading...</h1>;
	}

	if (token) {
		return (
			<Suspense fallback={<></>}>
				<Switch>
					<Route path="/app" exact>
						<Dashboard />
					</Route>
				</Switch>
			</Suspense>
		);
	}

	return (
		<Suspense fallback={<></>}>
			<Switch>
				<Route path="/app" exact>
					<Login />
				</Route>
				<Route path="/app/reset-password">
					<ResetPassword />
				</Route>
			</Switch>
		</Suspense>
	);
}
