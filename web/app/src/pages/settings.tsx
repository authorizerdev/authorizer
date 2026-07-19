import React, { useEffect, useState } from 'react';
import {
	AuthorizerMFASetup,
	useAuthorizer,
} from '@authorizerdev/authorizer-react';
import { Link } from 'react-router-dom';

export default function Settings() {
	const { user, config, authorizerRef } = useAuthorizer();
	// AuthorizerMFASetup only knows what the caller tells it - there's no
	// per-user enrolment signal for TOTP/email-OTP/SMS-OTP, but passkeys can
	// be checked directly so the Passkey row can highlight as already set up.
	const [passkeyRegistered, setPasskeyRegistered] = useState(false);

	useEffect(() => {
		authorizerRef.webauthnCredentials().then(({ data, errors }) => {
			if (!errors?.length && data) {
				setPasskeyRegistered(data.length > 0);
			}
		});
	}, [authorizerRef]);

	return (
		<div>
			<h1 style={{ textAlign: 'center' }}>Multi-factor authentication</h1>
			<p>
				Signed in as{' '}
				<a href={`mailto:${user?.email}`} style={{ color: '#3B82F6' }}>
					{user?.email}
				</a>
				. Set up an additional sign-in method to secure your account.
			</p>
			<br />
			<AuthorizerMFASetup
				availableMfaMethods={{
					totp: config.is_totp_mfa_enabled,
					passkey: config.is_webauthn_enabled,
					emailOtp: config.is_email_otp_mfa_enabled,
					smsOtp: config.is_sms_otp_mfa_enabled,
				}}
				heading="Add a second step to sign in"
				passkeyRegistered={passkeyRegistered}
			/>
			<br />
			<div style={{ textAlign: 'center' }}>
				<Link to="/app" style={{ color: '#3B82F6' }}>
					Back to dashboard
				</Link>
			</div>
		</div>
	);
}
