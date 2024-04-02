import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { AuthorizerProvider } from '@authorizerdev/authorizer-react';
import Root from './Root';
import { createRandomString } from './utils/common';

declare global {
	interface Window {
		__authorizer__: any;
	}
}

export default function App() {
	const searchParams = new URLSearchParams(window.location.search);
	const state = searchParams.get('state') || createRandomString();
	const scope = searchParams.get('scope')
		? searchParams.get('scope')?.toString().split(' ')
		: `openid profile email`;

	const urlProps: Record<string, any> = {
		state,
		scope,
	};

	const redirectURL =
		searchParams.get('redirect_uri') || searchParams.get('redirectURL');
	if (redirectURL) {
		urlProps.redirectURL = redirectURL;
	} else {
		urlProps.redirectURL = window.location.href;
	}
	const globalState: Record<string, string> = {
		...window['__authorizer__'],
		...urlProps,
	};
	return (
		<div
			style={{
				display: 'flex',
				justifyContent: 'center',
				flexDirection: 'column',
			}}
		>
			<div
				style={{
					display: 'flex',
					justifyContent: 'center',
					marginTop: 20,
					flexDirection: 'column',
					alignItems: 'center',
				}}
			>
				<img
					src={`${globalState.organizationLogo}`}
					alt="logo"
					style={{ height: 60, objectFit: 'cover' }}
				/>
				<h1>{globalState.organizationName}</h1>
			</div>
			<div className="container">
				<BrowserRouter>
					<AuthorizerProvider
						config={{
							authorizerURL: window.location.origin,
							redirectURL: globalState.redirectURL,
						}}
					>
						<Root globalState={globalState} />
					</AuthorizerProvider>
				</BrowserRouter>
			</div>
		</div>
	);
}
