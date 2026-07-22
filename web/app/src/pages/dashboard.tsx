import React from 'react';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import { Link } from 'react-router-dom';

export default function Dashboard() {
	const [loading, setLoading] = React.useState(false);
	const { user, setToken, authorizerRef } = useAuthorizer();

	const onLogout = async () => {
		setLoading(true);
		try {
			await authorizerRef.logout();
		} finally {
			// Always clear the local session and drop the loading state, even if
			// the server logout call fails — the user is logged out client-side
			// regardless, so the UI must never get stuck on "Processing....".
			setToken(null);
			setLoading(false);
		}
	};

	return (
		<div>
			<h1 className="au-page-title">Hey 👋</h1>
			<p className="au-center au-muted">Thank you for using Authorizer.</p>
			<p className="au-center">
				Signed in as <a href={`mailto:${user?.email}`}>{user?.email}</a>
			</p>
			<p className="au-center">
				<Link className="au-link" to="/app/settings">
					Manage MFA
				</Link>
			</p>

			<br />
			<button
				type="button"
				className="styled-button"
				style={{
					width: '100%',
					backgroundColor: loading
						? 'var(--authorizer-primary-disabled-color)'
						: 'var(--authorizer-white-color)',
					color: 'var(--authorizer-text-color)',
					border: '1px',
				}}
				disabled={loading}
				onClick={onLogout}
			>
				{loading ? 'Logging out...' : 'Logout'}
			</button>
		</div>
	);
}
