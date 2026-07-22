import React, { Fragment, useState } from 'react';
import {
	AuthorizerBasicAuthLogin,
	AuthorizerForgotPassword,
	AuthorizerMagicLinkLogin,
	AuthorizerPasskeyLogin,
	AuthorizerSocialLogin,
	useAuthorizer,
} from '@authorizerdev/authorizer-react';
import styled from 'styled-components';
import { Link, useLocation } from 'react-router-dom';

const enum VIEW_TYPES {
	LOGIN = 'login',
	FORGOT_PASSWORD = 'forgot-password',
}

const Footer = styled.div`
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
	margin-top: 15px;
`;

const FooterContent = styled.div`
	display: flex;
	justify-content: center;
	align-items: center;
	margin-top: 10px;
`;

const HRDForm = styled.form`
	display: flex;
	flex-direction: column;
	width: 100%;
`;

// Visually hidden but still exposed to screen readers (WCAG 3.3.2) — the HRD
// email field has only a placeholder otherwise, which assistive tech ignores.
const VisuallyHiddenLabel = styled.label`
	position: absolute;
	width: 1px;
	height: 1px;
	padding: 0;
	margin: -1px;
	overflow: hidden;
	clip: rect(0, 0, 0, 0);
	white-space: nowrap;
	border: 0;
`;

const HRDInput = styled.input`
	width: 100%;
	padding: 10px;
	margin-bottom: 10px;
	box-sizing: border-box;
	border: 1px solid #d1d5db;
	border-radius: 5px;
`;

const HRDButton = styled.button`
	width: 100%;
	padding: 10px;
	border: none;
	border-radius: 5px;
	cursor: pointer;
`;

// homeRealmDiscovery asks the server which enterprise SSO connection (if any) a
// login email's verified domain should route to. Returns the SP-initiated login
// URL to redirect to, or null when there is no SSO match (or discovery is
// disabled / errors) — in which case the caller falls back to the standard
// password / social / magic-link UI. Routing hint only; never blocks login.
async function homeRealmDiscovery(
	email: string,
): Promise<{ type: string; login_url: string } | null> {
	try {
		const res = await fetch(
			`${window.location.origin}/api/v1/org-discovery?email=${encodeURIComponent(
				email,
			)}`,
		);
		if (!res.ok) {
			return null;
		}
		const data = await res.json();
		if (data && data.connection && data.connection.login_url) {
			return data.connection;
		}
	} catch {
		// Discovery is a best-effort routing hint; any failure falls back to the
		// standard login UI so a user is never locked out.
	}
	return null;
}

export default function Login({ urlProps }: { urlProps: Record<string, any> }) {
	const { config } = useAuthorizer();
	// Preserved on the Sign Up link below: dropping the query string here
	// (state/client_id/redirect_uri/...) strands a user who signs up
	// mid-OAuth-flow on the dashboard with no way back to /authorize, since
	// Root.tsx's resumption effect reads these from the current URL only.
	const location = useLocation();
	const [view, setView] = useState<VIEW_TYPES>(VIEW_TYPES.LOGIN);
	// Email-first Home Realm Discovery: show an email field first. On an SSO
	// match we redirect to the org's SP-initiated login; otherwise we reveal the
	// standard login UI (ssoResolved). Falls open on any error/no-match.
	const [ssoResolved, setSsoResolved] = useState(false);
	const [hrdEmail, setHrdEmail] = useState('');
	const [hrdChecking, setHrdChecking] = useState(false);
	// AuthorizerPasskeyLogin and AuthorizerBasicAuthLogin each take over the
	// whole login surface once their own sign-in needs a second factor (their
	// own MFA setup/verify/locked screens) - every other login option, and
	// the login attempt not currently in flight, don't belong stacked on top
	// of those screens.
	const [passkeyStep, setPasskeyStep] = useState<
		'button' | 'mfa-setup' | 'mfa-verify' | 'locked'
	>('button');
	const [basicAuthStep, setBasicAuthStep] = useState<
		'form' | 'mfa-setup' | 'otp-verify' | 'locked'
	>('form');
	const passkeyIdle = passkeyStep === 'button';
	const basicAuthIdle = basicAuthStep === 'form';
	// Social login, magic link, and the forgot-password/sign-up footers only
	// make sense while both login attempts are idle.
	const showChrome = passkeyIdle && basicAuthIdle;

	const handleHRDSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		const email = hrdEmail.trim();
		if (!email) {
			setSsoResolved(true);
			return;
		}
		setHrdChecking(true);
		const connection = await homeRealmDiscovery(email);
		if (connection) {
			// Append the caller's OAuth context: the SAML/OIDC SP login endpoints
			// consume redirect_uri + state and thread them through the IdP
			// round-trip so the flow returns to the original caller, not /app.
			const params = new URLSearchParams();
			const redirectURI = urlProps.redirect_uri || urlProps.redirectURL;
			if (redirectURI) params.set('redirect_uri', redirectURI);
			if (urlProps.state) params.set('state', urlProps.state);
			window.location.assign(
				`${window.location.origin}${connection.login_url}?${params.toString()}`,
			);
			return;
		}
		// No SSO match → standard login UI.
		setHrdChecking(false);
		setSsoResolved(true);
	};

	// Email-first is opt-in per deployment: only when org discovery is enabled
	// (server-injected flag mirroring Meta.is_org_discovery_enabled). Off →
	// render today's password/social/magic-link UI directly, zero regression.
	if (
		urlProps.isOrgDiscoveryEnabled &&
		view === VIEW_TYPES.LOGIN &&
		!ssoResolved
	) {
		return (
			<Fragment>
				<h1 style={{ textAlign: 'center' }}>Login</h1>
				<HRDForm onSubmit={handleHRDSubmit}>
					<VisuallyHiddenLabel htmlFor="hrd-email">Email</VisuallyHiddenLabel>
					<HRDInput
						id="hrd-email"
						type="email"
						placeholder="Enter your email"
						value={hrdEmail}
						onChange={(e) => setHrdEmail(e.target.value)}
						autoFocus
					/>
					<HRDButton type="submit" disabled={hrdChecking}>
						{hrdChecking ? 'Checking...' : 'Continue'}
					</HRDButton>
				</HRDForm>
				<Footer>
					<Link
						to="#"
						onClick={() => setSsoResolved(true)}
						style={{ marginTop: 10 }}
					>
						Use another login method
					</Link>
				</Footer>
			</Fragment>
		);
	}

	return (
		<Fragment>
			{view === VIEW_TYPES.LOGIN && (
				<Fragment>
					<h1 style={{ textAlign: 'center' }}>Login</h1>
					{showChrome && <AuthorizerSocialLogin urlProps={urlProps} />}
					{basicAuthIdle && (
						<AuthorizerPasskeyLogin onStepChange={setPasskeyStep} />
					)}
					{passkeyIdle && (
						<Fragment>
							<br />
							{(config.is_basic_authentication_enabled ||
								config.is_mobile_basic_authentication_enabled) &&
								!config.is_magic_link_login_enabled && (
									<AuthorizerBasicAuthLogin
										urlProps={urlProps}
										onStepChange={setBasicAuthStep}
									/>
								)}
							{showChrome && config.is_magic_link_login_enabled && (
								<AuthorizerMagicLinkLogin urlProps={urlProps} />
							)}
							{showChrome &&
								(config.is_basic_authentication_enabled ||
									config.is_mobile_basic_authentication_enabled) &&
								!config.is_magic_link_login_enabled && (
									<Footer>
										<Link
											to="#"
											onClick={() => setView(VIEW_TYPES.FORGOT_PASSWORD)}
											style={{ marginBottom: 10 }}
										>
											Forgot Password?
										</Link>
									</Footer>
								)}
						</Fragment>
					)}
				</Fragment>
			)}
			{view === VIEW_TYPES.FORGOT_PASSWORD && (
				<Fragment>
					<h1 style={{ textAlign: 'center' }}>Forgot Password</h1>
					<AuthorizerForgotPassword
						urlProps={{
							...urlProps,
							redirect_uri: `${window.location.origin}/app/reset-password`,
						}}
						onPasswordReset={() => {
							setView(VIEW_TYPES.LOGIN);
						}}
					/>
					<Footer>
						<Link
							to="#"
							onClick={() => setView(VIEW_TYPES.LOGIN)}
							style={{ marginBottom: 10 }}
						>
							Back
						</Link>
					</Footer>
				</Fragment>
			)}
			{showChrome &&
				config.is_basic_authentication_enabled &&
				!config.is_magic_link_login_enabled &&
				config.is_sign_up_enabled && (
					<FooterContent>
						Don't have an account? &nbsp;{' '}
						<Link to={{ pathname: '/app/signup', search: location.search }}>
							{' '}
							Sign Up
						</Link>
					</FooterContent>
				)}
		</Fragment>
	);
}
