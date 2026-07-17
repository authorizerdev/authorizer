import React, { Fragment } from 'react';
import {
	AuthorizerSignup,
	AuthorizerSocialLogin,
} from '@authorizerdev/authorizer-react';
import styled from 'styled-components';
import { Link, useLocation } from 'react-router-dom';

const FooterContent = styled.div`
	display: flex;
	justify-content: center;
	align-items: center;
	margin-top: 20px;
`;

export default function SignUp({
	urlProps,
}: {
	urlProps: Record<string, any>;
}) {
	// Preserved on the Login link below - same reasoning as login.tsx's
	// Sign Up link: dropping the OAuth query string strands the user.
	const location = useLocation();
	return (
		<Fragment>
			<h1 style={{ textAlign: 'center' }}>Sign Up</h1>
			<br />
			<AuthorizerSocialLogin urlProps={urlProps} />
			<AuthorizerSignup urlProps={urlProps} />
			<FooterContent>
				Already have an account? <Link to={{ pathname: '/app', search: location.search }}> Login</Link>
			</FooterContent>
		</Fragment>
	);
}
