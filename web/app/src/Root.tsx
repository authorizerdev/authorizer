import { useEffect, lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import SetupPassword from './pages/setup-password';
import { hasWindow, createRandomString } from './utils/common';

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));
const SignUp = lazy(() => import('./pages/signup'));

export default function Root({
	globalState,
}: {
	globalState: Record<string, string>;
}) {
	const { token, loading, config } = useAuthorizer();

	const searchParams = new URLSearchParams(
		hasWindow() ? window.location.search : ``,
	);
	const state = searchParams.get('state') || createRandomString();
	const scope = searchParams.get('scope')
		? searchParams.get('scope')?.toString().split(' ')
		: ['openid', 'profile', 'email'];
	const code = searchParams.get('code') || '';
	const nonce = searchParams.get('nonce') || '';

	const urlProps: Record<string, any> = {
		state,
		scope,
	};

	const redirectURL =
		searchParams.get('redirect_uri') || searchParams.get('redirectURL');
	if (redirectURL) {
		urlProps.redirectURL = redirectURL;
	} else {
		urlProps.redirectURL = hasWindow() ? window.location.origin : redirectURL;
	}

	urlProps.redirect_uri = urlProps.redirectURL;

	useEffect(() => {
		if (token) {
			let redirectURL = config.redirectURL || '/app';
			// let params = `access_token=${token.access_token}&id_token=${token.id_token}&expires_in=${token.expires_in}&state=${globalState.state}`;
			// Note: If OIDC breaks in the future, use the above params
			let params = `state=${globalState.state}`;

			if (code !== '') {
				params += `&code=${code}`;
			}

			if (nonce !== '') {
				params += `&nonce=${nonce}`;
			}

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
	}, [token, config]);

	if (loading) {
		return <h1>Loading...</h1>;
	}

	if (token) {
		return (
			<Suspense fallback={<></>}>
				<Routes>
					<Route path="/app" element={<Dashboard />} />
				</Routes>
			</Suspense>
		);
	}

	return (
		<Suspense fallback={<></>}>
			<Routes>
				<Route path="/app" element={<Login urlProps={urlProps} />} />
				<Route path="/app/signup" element={<SignUp urlProps={urlProps} />} />
				<Route path="/app/reset-password" element={<ResetPassword />} />
				<Route path="/app/setup-password" element={<SetupPassword />} />
			</Routes>
		</Suspense>
	);
}
