import React, { useEffect, lazy, Suspense } from 'react';
import { Switch, Route } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import styled, { ThemeProvider } from 'styled-components';
import SetupPassword from './pages/setup-password';
import { hasWindow, createRandomString } from './utils/common';
import { theme } from './theme';

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));
const SignUp = lazy(() => import('./pages/signup'));

const Wrapper = styled.div`
	font-family: ${(props) => props.theme.fonts.fontStack};
	color: ${(props) => props.theme.colors.textColor};
	font-size: ${(props) => props.theme.fonts.mediumText};
	box-sizing: border-box;

	*,
	*:before,
	*:after {
		box-sizing: inherit;
	}
`;

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
	const code = searchParams.get('code') || ''
	const nonce = searchParams.get('nonce') || ''

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
			let params = `access_token=${token.access_token}&id_token=${token.id_token}&expires_in=${token.expires_in}&state=${globalState.state}`;

			if (code !== '') {
				params += `&code=${code}`
			}

			if (nonce !== '') {
				params += `&nonce=${nonce}`
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
			<ThemeProvider theme={theme}>
				<Wrapper>
					<Switch>
						<Route path="/app" exact>
							<Login urlProps={urlProps} />
						</Route>
						<Route path="/app/signup" exact>
							<SignUp urlProps={urlProps} />
						</Route>
						<Route path="/app/reset-password">
							<ResetPassword />
						</Route>
						<Route path="/app/setup-password">
							<SetupPassword />
						</Route>
					</Switch>
				</Wrapper>
			</ThemeProvider>
		</Suspense>
	);
}
