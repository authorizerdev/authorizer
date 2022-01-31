import React, { useEffect, lazy, Suspense } from 'react';
import { Switch, Route } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));

export default function Root() {
	const { token, loading, config } = useAuthorizer();

	console.log(config);
	useEffect(() => {
		if (token) {
			const url = new URL(config.redirectURL || '/app');
			console.log('=> url', url);
			if (url.origin !== window.location.origin) {
				window.location.href = config.redirectURL || '/app';
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
