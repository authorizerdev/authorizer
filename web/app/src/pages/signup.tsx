import React, { Fragment } from 'react';
import {
	AuthorizerSignup,
	AuthorizerSocialLogin,
} from '@authorizerdev/authorizer-react';
import styled from 'styled-components';
import { Link } from 'react-router-dom';

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
	return (
		<Fragment>
			<h1 style={{ textAlign: 'center' }}>Sign Up</h1>
			<br />
			<AuthorizerSocialLogin urlProps={urlProps} />
			<AuthorizerSignup urlProps={urlProps} />
			<FooterContent>
				Already have an account? <Link to="/app"> Login</Link>
			</FooterContent>
		</Fragment>
	);
}
