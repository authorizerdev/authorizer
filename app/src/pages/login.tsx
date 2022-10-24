import React, { Fragment, useState } from 'react';
import {
	AuthorizerBasicAuthLogin,
	AuthorizerForgotPassword,
	AuthorizerMagicLinkLogin,
	AuthorizerSocialLogin,
	useAuthorizer,
} from '@authorizerdev/authorizer-react';
import styled from 'styled-components';
import { Link } from 'react-router-dom';

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

export default function Login({ urlProps }: { urlProps: Record<string, any> }) {
	const { config } = useAuthorizer();
	const [view, setView] = useState<VIEW_TYPES>(VIEW_TYPES.LOGIN);
	return (
		<Fragment>
			{view === VIEW_TYPES.LOGIN && (
				<Fragment>
					<h1 style={{ textAlign: 'center' }}>Login</h1>
					<br />
					<AuthorizerSocialLogin urlProps={urlProps} />
					{config.is_basic_authentication_enabled &&
						!config.is_magic_link_login_enabled && (
							<AuthorizerBasicAuthLogin urlProps={urlProps} />
						)}
					{config.is_magic_link_login_enabled && (
						<AuthorizerMagicLinkLogin urlProps={urlProps} />
					)}
					<Footer>
						<Link
							to="#"
							onClick={() => setView(VIEW_TYPES.FORGOT_PASSWORD)}
							style={{ marginBottom: 10 }}
						>
							Forgot Password?
						</Link>
					</Footer>
				</Fragment>
			)}
			{view === VIEW_TYPES.FORGOT_PASSWORD && (
				<Fragment>
					<h1 style={{ textAlign: 'center' }}>Forgot Password</h1>
					<AuthorizerForgotPassword urlProps={{
						...urlProps,
						redirect_uri: `${window.location.origin}/app/reset-password`,
					}} />
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
			{config.is_basic_authentication_enabled &&
				!config.is_magic_link_login_enabled &&
				config.is_sign_up_enabled && (
					<FooterContent>
						Don't have an account? <Link to="/app/signup"> Sign Up</Link>
					</FooterContent>
				)}
		</Fragment>
	);
}
