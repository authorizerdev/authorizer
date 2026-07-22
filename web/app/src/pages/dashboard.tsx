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
			<h1>Hey 👋,</h1>
			<p>Thank you for using authorizer.</p>
			<p>
				Your email address is{' '}
				<a href={`mailto:${user?.email}`} style={{ color: '#3B82F6' }}>
					{user?.email}
				</a>
			</p>

			<p>
				<Link to="/app/settings" style={{ color: '#3B82F6' }}>
					Manage MFA
				</Link>
			</p>

			<br />
			{loading ? (
				<h3>Processing....</h3>
			) : (
				<button
					type="button"
					onClick={onLogout}
					style={{
						color: '#3B82F6',
						cursor: 'pointer',
						background: 'none',
						border: 'none',
						padding: 0,
						margin: '1em 0',
						font: 'inherit',
						fontSize: '1.17em',
						fontWeight: 'bold',
						display: 'block',
						textAlign: 'left',
					}}
				>
					Logout
				</button>
			)}
		</div>
	);
}
