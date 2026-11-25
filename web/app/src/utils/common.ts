export const getCrypto = () => {
	//ie 11.x uses msCrypto
	return (window.crypto || (window as any).msCrypto) as Crypto;
};

export const createRandomString = () => {
	const charset =
		'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_~.';
	let random = '';
	const randomValues = Array.from(
		getCrypto().getRandomValues(new Uint8Array(43)),
	);
	randomValues.forEach((v) => (random += charset[v % charset.length]));
	return random;
};

export const createQueryParams = (params: any) => {
	return Object.keys(params)
		.filter((k) => typeof params[k] !== 'undefined')
		.map((k) => encodeURIComponent(k) + '=' + encodeURIComponent(params[k]))
		.join('&');
};

export const hasWindow = (): boolean => typeof window !== 'undefined';

/**
 * True when the params carry an OAuth/OIDC /authorize request the login UI
 * should resume after sign-in (see Root.tsx). Shared with the reset-password
 * page so a password reset started mid-authorize returns to the same flow.
 */
export const isAuthorizeContextParams = (params: URLSearchParams): boolean =>
	(params.get('redirect_uri') || params.get('redirectURL') || '') !== '' &&
	['state', 'response_type', 'response_mode', 'client_id', 'scope'].some(
		(k) => (params.get(k) || '') !== '',
	);
