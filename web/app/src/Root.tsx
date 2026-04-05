import { useEffect, lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import SetupPassword from './pages/setup-password';
import { hasWindow, createRandomString } from './utils/common';

function isValidRedirectUri(uri: string): boolean {
	try {
		const url = new URL(uri, window.location.origin);
		if (url.origin === window.location.origin) return true;
		// Only allow http/https protocols to prevent javascript: or data: URIs
		if (url.protocol !== 'http:' && url.protocol !== 'https:') return false;
		return false;
	} catch {
		return false;
	}
}

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));
const SignUp = lazy(() => import('./pages/signup'));

/**
 * Validates a redirect URI to prevent open redirect attacks.
 * Allows same-origin redirects and cross-origin redirects only for
 * http/https protocols that match configured redirect URLs.
 */
function isValidRedirectUri(
	uri: string,
	configuredRedirectURL?: string,
): boolean {
	try {
		const url = new URL(uri, window.location.origin);
		// Only allow http and https protocols (block javascript:, data:, etc.)
		if (url.protocol !== 'http:' && url.protocol !== 'https:') {
			return false;
		}
		// Same-origin redirects are always allowed
		if (url.origin === window.location.origin) {
			return true;
		}
		// Cross-origin: only allow if it matches the configured redirect URL origin
		if (configuredRedirectURL) {
			try {
				const configuredUrl = new URL(configuredRedirectURL);
				if (url.origin === configuredUrl.origin) {
					return true;
				}
			} catch {
				// Invalid configured URL, reject cross-origin
			}
		}
		return false;
	} catch {
		// If URI can't be parsed, reject it
		return false;
	}
}

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

	const rawRedirectURL =
		searchParams.get('redirect_uri') || searchParams.get('redirectURL');
	if (rawRedirectURL && isValidRedirectUri(rawRedirectURL, config?.redirectURL)) {
		urlProps.redirectURL = rawRedirectURL;
	} else {
		urlProps.redirectURL = hasWindow() ? window.location.origin : '/';
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
				// Only allow safe protocols to prevent javascript: or data: URI attacks
				if (url.protocol === 'http:' || url.protocol === 'https:') {
					sessionStorage.removeItem('authorizer_state');
					window.location.replace(redirectURL);
				}
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
					<Route path="*" element={<Navigate to="/app" replace />} />
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
