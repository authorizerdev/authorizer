import { useEffect, lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import {
	AuthorizerMFASetup,
	AuthorizerVerifyOtp,
	useAuthorizer,
} from '@authorizerdev/authorizer-react';
import { parseMfaRedirectParams } from '@authorizerdev/authorizer-js';
import SetupPassword from './pages/setup-password';
import { hasWindow, createRandomString } from './utils/common';

const ResetPassword = lazy(() => import('./pages/rest-password'));
const Login = lazy(() => import('./pages/login'));
const Dashboard = lazy(() => import('./pages/dashboard'));
const Settings = lazy(() => import('./pages/settings'));
const SignUp = lazy(() => import('./pages/signup'));

/**
 * True only for the exact shape `bounceSAMLIDPToLogin` (saml_idp.go) produces:
 * same-origin, path `/saml/idp/{slug}/sso`, carrying `saml_continue`. This is
 * deliberately narrow — `redirect_uri` is a client-controllable query param on
 * /app, so anything looser here becomes a post-login open redirect.
 */
function isSamlIdpContinueURL(url: string): boolean {
	if (!hasWindow() || !url) return false;
	let parsed: URL;
	try {
		parsed = new URL(url, window.location.origin);
	} catch {
		return false;
	}
	return (
		parsed.origin === window.location.origin &&
		/^\/saml\/idp\/[^/]+\/sso$/.test(parsed.pathname) &&
		parsed.searchParams.has('saml_continue')
	);
}

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
	const { token, loading, config, setAuthData } = useAuthorizer();

	// The server redirects here with these params, instead of issuing a
	// token, when its MFA gate withholds one - not just for the OAuth
	// /authorize flow this originated for, but also the magic-link-login and
	// signup-email-verification click-through URLs (GET /verify_email),
	// which redirect to the same place with the same params.
	const mfaRedirect = hasWindow()
		? parseMfaRedirectParams(window.location.href)
		: null;

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
	urlProps.redirectURL =
		rawRedirectURL || (hasWindow() ? `${window.location.origin}/app` : '/app');

	urlProps.redirect_uri = urlProps.redirectURL;

	// Server-injected flag (window.__authorizer__) mirroring
	// Config.EnableOrgDiscovery / Meta.is_org_discovery_enabled. When false the
	// login page skips the email-first SSO step entirely (unchanged behavior).
	urlProps.isOrgDiscoveryEnabled =
		(globalState as Record<string, unknown>).isOrgDiscoveryEnabled === true;

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

	// Separate resumption mechanism: SP-initiated SAML IdP login. The server
	// (bounceSAMLIDPToLogin) sends unauthenticated users here with
	// redirect_uri pointing back at its own /saml/idp/{slug}/sso?saml_continue
	// endpoint, which resumes and auto-submits the pending assertion once a
	// session exists. Unlike the /authorize resumption above, we navigate to
	// the literal redirect_uri - but only when it matches that exact shape,
	// never for an arbitrary client-supplied redirect_uri.
	useEffect(() => {
		if (!token) return;
		if (!isSamlIdpContinueURL(rawRedirectURL)) return;
		window.location.replace(rawRedirectURL);
	}, [token, rawRedirectURL]);

	// Both MFA gates below are reached via a server redirect carrying the
	// gate state in the URL, not client-side navigation - there's no prior
	// SPA screen to pop back to. "Back" here means abandoning the pending
	// MFA session and returning to a clean /app (fresh login screen).
	const backToLogin = () => window.location.replace('/app');

	if (loading) {
		return <h1>Loading...</h1>;
	}
	if (mfaRedirect && mfaRedirect.mfaGate === 'verify') {
		// An already-configured factor must be challenged, not offered setup
		// again - no email/phone_number in hand (OAuth/magic-link return), but
		// verify_otp resolves the pending user from the MFA session cookie
		// alone, same as the passkey-primary-login continuation.
		return (
			<AuthorizerVerifyOtp
				is_totp={mfaRedirect.mfaMethods.includes('totp')}
				offerWebauthnVerify={mfaRedirect.mfaMethods.includes('webauthn')}
				hasCodeFactor={
					mfaRedirect.mfaMethods.includes('totp') ||
					mfaRedirect.mfaMethods.includes('email_otp') ||
					mfaRedirect.mfaMethods.includes('sms_otp')
				}
				onBack={backToLogin}
				onLogin={(data: any) => {
					setAuthData({
						user: data?.user || null,
						token: data,
						config,
						loading: false,
					});
				}}
			/>
		);
	}
	if (mfaRedirect && mfaRedirect.mfaGate === 'offer') {
		return (
			<AuthorizerMFASetup
				availableMfaMethods={{
					totp: mfaRedirect.mfaMethods.includes('totp'),
					passkey: mfaRedirect.mfaMethods.includes('webauthn'),
					emailOtp: mfaRedirect.mfaMethods.includes('email_otp'),
					smsOtp: mfaRedirect.mfaMethods.includes('sms_otp'),
				}}
				heading="Set up multi-factor authentication"
				onBack={backToLogin}
				loginContext={{
					onComplete: (data: any) => {
						setAuthData({
							user: data?.user || null,
							token: data,
							config,
							loading: false,
						});
					},
				}}
			/>
		);
	}
	if (token) {
		return (
			<Suspense fallback={<></>}>
				<Routes>
					<Route path="/app" element={<Dashboard />} />
					<Route path="/app/settings" element={<Settings />} />
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
