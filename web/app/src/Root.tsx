import { useEffect, useRef, lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import { Authorizer } from '@authorizerdev/authorizer-js';
import SetupPassword from './pages/setup-password';
import { hasWindow, createRandomString } from './utils/common';

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
		if (url.protocol !== 'http:' && url.protocol !== 'https:') {
			return false;
		}
		if (url.origin === window.location.origin) {
			return true;
		}
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
		return false;
	}
}

export default function Root({
	globalState,
}: {
	globalState: Record<string, string>;
}) {
	const { token, loading, config } = useAuthorizer();

	// Read params from both query string and fragment to support all response_modes.
	const queryParams = new URLSearchParams(
		hasWindow() ? window.location.search : ``,
	);
	const fragmentParams = new URLSearchParams(
		hasWindow() && window.location.hash
			? window.location.hash.substring(1)
			: ``,
	);
	const getParam = (key: string): string =>
		queryParams.get(key) || fragmentParams.get(key) || '';

	const state = getParam('state') || createRandomString();
	const scope = getParam('scope')
		? getParam('scope').split(' ')
		: ['openid', 'profile', 'email'];
	const code = getParam('code');
	const nonce = getParam('nonce');
	const responseType = getParam('response_type');
	const responseMode = getParam('response_mode');

	const searchParams = queryParams;

	const urlProps: Record<string, any> = {
		state,
		scope,
	};

	const rawRedirectURL =
		searchParams.get('redirect_uri') || searchParams.get('redirectURL');
	if (
		rawRedirectURL &&
		isValidRedirectUri(rawRedirectURL, config?.redirectURL)
	) {
		urlProps.redirectURL = rawRedirectURL;
	} else {
		urlProps.redirectURL = hasWindow() ? window.location.origin : '/';
	}

	urlProps.redirect_uri = urlProps.redirectURL;

	// For OIDC flows, prefer the redirect_uri from the URL (RP's callback)
	const oidcRedirectURI = rawRedirectURL || config.redirectURL || '/app';

	// Track whether the OIDC code state has been ensured
	const codeStateEnsured = useRef(false);

	useEffect(() => {
		if (!token) return;

		// Detect OIDC authorize flow: code and state in URL means we came
		// from /authorize and need to redirect back to the RP.
		const isOIDCFlow = code !== '' && getParam('state') !== '';

		if (isOIDCFlow && !codeStateEnsured.current) {
			codeStateEnsured.current = true;

			// Use authorizer-js client to call session with state parameter.
			// This ensures the authorization code state is stored in the backend
			// (needed when the SDK auto-detected an existing session and the
			// login mutation was never called).
			const authorizerClient = new Authorizer({
				authorizerURL: window.location.origin,
				redirectURL: oidcRedirectURI,
				clientID: globalState.clientId || config.client_id,
			});

			authorizerClient
				.getSession(undefined, { state: getParam('state'), scope } as any)
				.then((res) => {
					if (res?.data) {
						performRedirect(oidcRedirectURI, res.data);
					} else {
						// Session call failed, try with existing token
						performRedirect(oidcRedirectURI, token);
					}
				})
				.catch(() => {
					performRedirect(oidcRedirectURI, token);
				});
			return;
		}

		// Non-OIDC flow with a redirect target — redirect immediately
		if (rawRedirectURL && code !== '') {
			performRedirect(oidcRedirectURI, token);
		}

		return () => {};
	}, [token, config]);

	function performRedirect(
		baseRedirectURL: string,
		tokenData: Record<string, any>,
	) {
		if (!tokenData) return;
		let redirectURL = baseRedirectURL;

		let params = '';
		const isImplicit =
			responseType === 'token' ||
			responseType === 'id_token' ||
			responseType === 'id_token token';
		const isCodeFlow = responseType === 'code' || responseType === '';

		if (isCodeFlow) {
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (code !== '') {
				params += `&code=${encodeURIComponent(code)}`;
			}
		} else if (isImplicit) {
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (
				tokenData.access_token &&
				(responseType === 'token' || responseType === 'id_token token')
			) {
				params += `&access_token=${encodeURIComponent(tokenData.access_token)}`;
				params += `&token_type=Bearer`;
				if (tokenData.expires_in) {
					params += `&expires_in=${tokenData.expires_in}`;
				}
			}
			if (
				tokenData.id_token &&
				(responseType === 'id_token' || responseType === 'id_token token')
			) {
				params += `&id_token=${encodeURIComponent(tokenData.id_token)}`;
			}
			if (nonce !== '') {
				params += `&nonce=${encodeURIComponent(nonce)}`;
			}
		} else if (responseType.includes('code')) {
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (code !== '') {
				params += `&code=${encodeURIComponent(code)}`;
			}
			if (
				tokenData.access_token &&
				responseType.includes('token') &&
				!responseType.startsWith('id_token')
			) {
				params += `&access_token=${encodeURIComponent(tokenData.access_token)}`;
				params += `&token_type=Bearer`;
				if (tokenData.expires_in) {
					params += `&expires_in=${tokenData.expires_in}`;
				}
			}
			if (tokenData.id_token && responseType.includes('id_token')) {
				params += `&id_token=${encodeURIComponent(tokenData.id_token)}`;
			}
			if (nonce !== '') {
				params += `&nonce=${encodeURIComponent(nonce)}`;
			}
		} else {
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (code !== '') {
				params += `&code=${encodeURIComponent(code)}`;
			}
		}

		const useFragment =
			responseMode === 'fragment' ||
			(isImplicit && responseMode !== 'query' && responseMode !== 'form_post');

		try {
			const url = new URL(redirectURL);
			if (useFragment) {
				redirectURL = redirectURL.split('#')[0] + '#' + params;
			} else {
				if (redirectURL.includes('?')) {
					redirectURL = `${redirectURL}&${params}`;
				} else {
					redirectURL = `${redirectURL}?${params}`;
				}
			}

			if (
				url.protocol === 'http:' ||
				url.protocol === 'https:' ||
				url.origin === window.location.origin
			) {
				sessionStorage.removeItem('authorizer_state');
				window.location.replace(redirectURL);
			}
		} catch {
			if (redirectURL.startsWith('/')) {
				window.location.replace(redirectURL);
			}
		}
	}

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
