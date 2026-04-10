import { useEffect, lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import SetupPassword from './pages/setup-password';
import { hasWindow, createRandomString } from './utils/common';

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));
const SignUp = lazy(() => import('./pages/signup'));

/**
 * Build a normalized parameter map from query + fragment.
 * We treat both as inputs because `/authorize` may choose fragment
 * depending on response_mode and our login UI should preserve the
 * original request context exactly.
 */
function getCombinedParams(): URLSearchParams {
	const queryParams = new URLSearchParams(
		hasWindow() ? window.location.search : ``,
	);
	const fragmentParams = new URLSearchParams(
		hasWindow() && window.location.hash
			? window.location.hash.substring(1)
			: ``,
	);

	// Query takes precedence over fragment when both exist.
	const combined = new URLSearchParams(fragmentParams);
	for (const [k, v] of queryParams.entries()) {
		combined.set(k, v);
	}
	return combined;
}

export default function Root({
	globalState,
}: {
	globalState: Record<string, string>;
}) {
	const { token, loading, config } = useAuthorizer();

	const combinedParams = getCombinedParams();
	const getParam = (key: string): string => combinedParams.get(key) || '';

	const state = getParam('state') || createRandomString();
	const scope = getParam('scope')
		? getParam('scope').split(' ')
		: ['openid', 'profile', 'email'];
	const nonce = getParam('nonce');
	const responseType = getParam('response_type');
	const responseMode = getParam('response_mode');

	const urlProps: Record<string, any> = {
		state,
		scope,
	};

	const rawRedirectURL = getParam('redirect_uri') || getParam('redirectURL');
	urlProps.redirectURL = rawRedirectURL || (hasWindow() ? window.location.origin : '/app');

	urlProps.redirect_uri = urlProps.redirectURL;

	const isAuthorizeContext =
		rawRedirectURL !== '' &&
		(getParam('state') !== '' ||
			getParam('response_type') !== '' ||
			getParam('response_mode') !== '' ||
			getParam('client_id') !== '' ||
			getParam('scope') !== '');

	useEffect(() => {
		if (!token) return;

		// Security + correctness: the server `/authorize` endpoint is the
		// source of truth for redirect_uri validation and response_mode
		// (query / fragment / form_post / web_message). The login UI should
		// only establish a session and then resume the authorization request.
		if (!isAuthorizeContext) return;

		// Preserve exactly what we received on /app and send it back to
		// /authorize; the backend will complete the authorization response.
		const params = new URLSearchParams();
		for (const [k, v] of combinedParams.entries()) {
			// Ignore any accidental app-only params.
			if (k === '') continue;
			params.set(k, v);
		}

		// Ensure state exists; do NOT overwrite if provided.
		if (!params.get('state')) {
			params.set('state', state);
		}
		if (scope?.length && !params.get('scope')) {
			params.set('scope', scope.join(' '));
		}

		sessionStorage.removeItem('authorizer_state');
		window.location.replace(`/authorize?${params.toString()}`);
	}, [token, isAuthorizeContext, state]);

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
