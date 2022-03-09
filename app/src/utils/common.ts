export const getCrypto = () => {
	//ie 11.x uses msCrypto
	return (window.crypto || (window as any).msCrypto) as Crypto;
};

export const createRandomString = () => {
	const charset =
		'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_~.';
	let random = '';
	const randomValues = Array.from(
		getCrypto().getRandomValues(new Uint8Array(43))
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
