import {
	AuthorizerMFASetup,
	useAuthorizer,
} from '@authorizerdev/authorizer-react';
import { Link } from 'react-router-dom';

export default function Settings() {
	const { user, config } = useAuthorizer();

	return (
		<div>
			<h1 className="au-page-title">Multi-factor authentication</h1>
			<p className="au-center au-muted">
				Signed in as <a href={`mailto:${user?.email}`}>{user?.email}</a>. Set up
				an additional sign-in method to secure your account.
			</p>
			<br />
			<AuthorizerMFASetup
				availableMfaMethods={{
					totp: config.is_totp_mfa_enabled,
					passkey: config.is_webauthn_enabled,
					emailOtp: config.is_email_otp_mfa_enabled,
					smsOtp: config.is_sms_otp_mfa_enabled,
				}}
				// What this user already enrolled (server-side truth, part of the
				// user fragment) - those tiles render as "Enabled"/"Manage" instead
				// of offering a fresh setup.
				enrolledMethods={user?.enrolled_mfa_methods}
				heading="Add a second step to sign in"
			/>
			<br />
			<div className="au-center">
				<Link className="au-link" to="/app">
					Back to dashboard
				</Link>
			</div>
		</div>
	);
}
