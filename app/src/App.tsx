import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { AuthorizerProvider } from '@authorizerdev/authorizer-react';
import Root from './Root';

export default function App() {
	// @ts-ignore
	const globalState: Record<string, string> = window['__authorizer__'];
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
					style={{ height: 60, width: 60, objectFit: 'cover' }}
				/>
				<h1>{globalState.organizationName}</h1>
			</div>
			<div
				style={{
					width: 400,
					margin: `10px auto`,
					border: `1px solid #D1D5DB`,
					padding: `25px 20px`,
					borderRadius: 5,
				}}
			>
				<BrowserRouter>
					<AuthorizerProvider
						config={{
<<<<<<< HEAD
							authorizerURL: window.location.origin,
=======
							authorizerURL: globalState.authorizerURL,
>>>>>>> main
							redirectURL: globalState.redirectURL,
						}}
					>
						<Root />
					</AuthorizerProvider>
				</BrowserRouter>
			</div>
		</div>
	);
}
