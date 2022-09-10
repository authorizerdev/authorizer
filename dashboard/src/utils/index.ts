import _ from 'lodash';

export const hasAdminSecret = () => {
	return (<any>window)['__authorizer__'].isOnboardingCompleted === true;
};

export const capitalizeFirstLetter = (data: string): string =>
	data.charAt(0).toUpperCase() + data.slice(1);

const fallbackCopyTextToClipboard = (text: string) => {
	const textArea = document.createElement('textarea');

	textArea.value = text;
	textArea.style.top = '0';
	textArea.style.left = '0';
	textArea.style.position = 'fixed';

	document.body.appendChild(textArea);
	textArea.focus();
	textArea.select();

	try {
		const successful = document.execCommand('copy');
		const msg = successful ? 'successful' : 'unsuccessful';
		console.log('Fallback: Copying text command was ' + msg);
	} catch (err) {
		console.error('Fallback: Oops, unable to copy', err);
	}
	document.body.removeChild(textArea);
};

export const copyTextToClipboard = async (text: string) => {
	if (!navigator.clipboard) {
		fallbackCopyTextToClipboard(text);
		return;
	}
	try {
		navigator.clipboard.writeText(text);
	} catch (err) {
		throw err;
	}
};

export const getObjectDiff = (obj1: any, obj2: any) => {
	const diff = Object.keys(obj1).reduce((result, key) => {
		if (!obj2.hasOwnProperty(key)) {
			result.push(key);
		} else if (
			_.isEqual(obj1[key], obj2[key]) ||
			(obj1[key] === null && obj2[key] === '') ||
			(obj1[key] &&
				Array.isArray(obj1[key]) &&
				obj1[key].length === 0 &&
				obj2[key] === null)
		) {
			const resultKeyIndex = result.indexOf(key);
			result.splice(resultKeyIndex, 1);
		}
		return result;
	}, Object.keys(obj2));

	return diff;
};

export const validateEmail = (email: string) => {
	if (!email || email === '') return true;
	return email
		.toLowerCase()
		.match(
			/^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/
		)
		? true
		: false;
};

export const validateURI = (uri: string) => {
	if (!uri || uri === '') return true;
	return uri
		.toLowerCase()
		.match(
			/(?:^|\s)((https?:\/\/)?(?:localhost|[\w-]+(?:\.[\w-]+)+)(:\d+)?(\/\S*)?)/
		)
		? true
		: false;
};
