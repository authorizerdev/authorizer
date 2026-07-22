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
	const queryParams = new URLSearchParams(window.location.search);
	const fragmentParams = new URLSearchParams(
		window.location.hash ? window.location.hash.substring(1) : ``,
	);
	const getParam = (key: string): string =>
		queryParams.get(key) || fragmentParams.get(key) || '';

	const state = getParam('state') || createRandomString();
	const scope = getParam('scope')
		? getParam('scope').toString().split(' ')
		: ['openid', 'profile', 'email'];

	const urlProps: Record<string, any> = {
		state,
		scope,
	};

	const redirectURL = getParam('redirect_uri') || getParam('redirectURL');
	if (redirectURL) {
		urlProps.redirectURL = redirectURL;
	} else {
		urlProps.redirectURL = `${window.location.origin}/app`;
	}
	const globalState: Record<string, string> = {
		...window['__authorizer__'],
		...urlProps,
	};
	return (
		<div className="app-shell">
			<header className="app-brand">
				{globalState.organizationLogo && (
					<img src={globalState.organizationLogo} alt="logo" />
				)}
				<h1>{globalState.organizationName}</h1>
			</header>
			<main className="container">
				<BrowserRouter>
					<AuthorizerProvider
						config={{
							authorizerURL: window.location.origin,
							redirectURL: globalState.redirectURL,
							clientID: globalState.clientId,
						}}
					>
						<Root globalState={globalState} />
					</AuthorizerProvider>
				</BrowserRouter>
			</main>
		</div>
	);
}
