import React from 'react';
import {
	AuthorizerPasskeyRegister,
	useAuthorizer,
} from '@authorizerdev/authorizer-react';
import { Link } from 'react-router-dom';

export default function Settings() {
	const { user } = useAuthorizer();

	return (
		<div>
			<h1 style={{ textAlign: 'center' }}>Passkeys</h1>
			<p>
				Signed in as{' '}
				<a href={`mailto:${user?.email}`} style={{ color: '#3B82F6' }}>
					{user?.email}
				</a>
				. Add a passkey to sign in without a password next time.
			</p>
			<br />
			<AuthorizerPasskeyRegister showCredentials />
			<br />
			<div style={{ textAlign: 'center' }}>
				<Link to="/app" style={{ color: '#3B82F6' }}>
					Back to dashboard
				</Link>
			</div>
		</div>
	);
}
