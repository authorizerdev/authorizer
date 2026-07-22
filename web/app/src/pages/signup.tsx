import { Fragment, useState } from 'react';
import { AuthorizerSignup } from '@authorizerdev/authorizer-react';
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
	// AuthorizerSignup shows its own screens internally (MFA setup, OTP
	// verify, locked-out) once the account is created - "Already have an
	// account? Login" only makes sense above the initial form, not stacked on
	// top of those follow-up screens (a user mid-MFA-setup already has an
	// account and isn't signing up again).
	const [isBaseForm, setIsBaseForm] = useState(true);
	return (
		<Fragment>
			<h1 className="au-page-title">Sign Up</h1>
			<br />
			<AuthorizerSignup
				urlProps={urlProps}
				onStepChange={(step) => setIsBaseForm(step === 'form')}
			/>
			{isBaseForm && (
				<FooterContent>
					Already have an account? &nbsp;
					<Link to={{ pathname: '/app', search: location.search }}> Login</Link>
				</FooterContent>
			)}
		</Fragment>
	);
}
