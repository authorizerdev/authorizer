import { useEffect, useRef, lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
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

	// Read params from both query string and fragment to support all response_modes.
	// The /authorize handler may deliver params via ?query or #fragment depending
	// on the response_mode requested by the RP.
	const queryParams = new URLSearchParams(
		hasWindow() ? window.location.search : ``,
	);
	const fragmentParams = new URLSearchParams(
		hasWindow() && window.location.hash
			? window.location.hash.substring(1)
			: ``,
	);
	// Prefer query params, fall back to fragment params
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

	// Track whether we've already ensured the code state is stored.
	const codeStateEnsured = useRef(false);

	// Resolve the RP's redirect_uri: prefer the URL param (from /authorize),
	// fall back to the SDK config, and finally to '/app' for non-OIDC flows.
	const oidcRedirectURI = rawRedirectURL || config.redirectURL || '/app';

	useEffect(() => {
		if (!token) return;

		// When the SDK auto-detects a session during an OIDC /authorize flow
		// (code is in URL), the login mutation was never called, so the
		// authorization code state was never stored. We must call the session
		// GraphQL query WITH the state parameter so the backend stores the
		// code state before we redirect to the RP.
		const isOIDCFlow = code !== '' && getParam('state') !== '';
		if (isOIDCFlow && !codeStateEnsured.current) {
			codeStateEnsured.current = true;
			fetch('/graphql', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({
					query: `query session($params: SessionQueryRequest) { session(params: $params) { access_token id_token expires_in refresh_token } }`,
					variables: { params: { state: getParam('state'), scope } },
					operationName: 'session',
				}),
			})
				.then((res) => res.json())
				.then((res) => {
					if (res?.data?.session) {
						performRedirect(oidcRedirectURI, res.data.session);
					}
				})
				.catch(() => {
					performRedirect(oidcRedirectURI, token);
				});
			return;
		}

		// Non-OIDC flow or code state already ensured — redirect immediately
		if (code !== '' || rawRedirectURL) {
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

		// Build response params based on the response_type from the /authorize request.
			// RFC 6749 / OIDC Core: the redirect must include exactly the params
			// expected by the RP for the requested flow.
			let params = '';
			const isImplicit =
				responseType === 'token' ||
				responseType === 'id_token' ||
				responseType === 'id_token token';
			const isCodeFlow = responseType === 'code' || responseType === '';

		if (isCodeFlow) {
			// Authorization Code flow: return code + state only.
			// Tokens are exchanged at /oauth/token by the RP's backend.
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (code !== '') {
				params += `&code=${encodeURIComponent(code)}`;
			}
		} else if (isImplicit) {
			// Implicit flow: return tokens directly.
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
			// Hybrid flow (code id_token, code token, code id_token token):
			// return code + relevant tokens.
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (code !== '') {
				params += `&code=${encodeURIComponent(code)}`;
			}
			if (
				tokenData.access_token &&
				(responseType.includes('token') && !responseType.startsWith('id_token'))
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
			// Fallback: send state + code (backward compat)
			params = `state=${encodeURIComponent(globalState.state)}`;
			if (code !== '') {
				params += `&code=${encodeURIComponent(code)}`;
			}
		}

		// Determine delivery mode per OIDC spec:
		// - response_mode=query or code flow default → query string (?params)
		// - response_mode=fragment or implicit/hybrid default → fragment (#params)
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

			if (url.origin !== window.location.origin) {
				if (url.protocol === 'http:' || url.protocol === 'https:') {
					sessionStorage.removeItem('authorizer_state');
					window.location.replace(redirectURL);
				}
			} else {
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
