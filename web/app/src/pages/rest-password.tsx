import { Fragment } from 'react';
import { AuthorizerResetPassword } from '@authorizerdev/authorizer-react';

export default function ResetPassword() {
	return (
		<Fragment>
			<h1 className="au-page-title">Reset Password</h1>
			<br />
			<AuthorizerResetPassword />
		</Fragment>
	);
}
