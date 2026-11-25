import { Fragment } from 'react';
import { AuthorizerResetPassword } from '@authorizerdev/authorizer-react';
import { hasWindow, isAuthorizeContextParams } from '../utils/common';

/**
 * When the reset was started from a login page reached via /authorize, the
 * reset link carries that request's params (issue #344). After the reset,
 * send the user back to the same-origin login page with them (token
 * stripped) so signing in resumes /authorize, which re-validates
 * redirect_uri server-side. Navigation target is always our own /app, never a
 * URL taken from the query, so this cannot become an open redirect.
 */
function getAuthorizeResumeURL(): string | null {
	if (!hasWindow()) return null;
	const params = new URLSearchParams(window.location.search);
	params.delete('token');
	if (!isAuthorizeContextParams(params)) return null;
	return `/app?${params.toString()}`;
}

export default function ResetPassword() {
	const resumeURL = getAuthorizeResumeURL();
	return (
		<Fragment>
			<h1 className="au-page-title">Reset Password</h1>
			<br />
			<AuthorizerResetPassword
				onReset={
					resumeURL ? () => window.location.replace(resumeURL) : undefined
				}
			/>
		</Fragment>
	);
}
