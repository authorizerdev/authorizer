import { Fragment } from 'react';
import { AuthorizerResetPassword } from '@authorizerdev/authorizer-react';

export default function SetupPassword() {
	return (
		<Fragment>
			<h1 className="au-page-title">Setup new Password</h1>
			<br />
			<AuthorizerResetPassword />
		</Fragment>
	);
}
