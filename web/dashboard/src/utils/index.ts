import _ from 'lodash';

interface AuthorizerWindow extends Window {
	__authorizer__: {
		isOnboardingCompleted: boolean;
	};
}

export const hasAdminSecret = () => {
	return (window as unknown as AuthorizerWindow).__authorizer__
		.isOnboardingCompleted === true;
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
		document.execCommand('copy');
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
		await navigator.clipboard.writeText(text);
	} catch (err) {
		throw err;
	}
};

export const getObjectDiff = (
	obj1: Record<string, unknown>,
	obj2: Record<string, unknown>,
): string[] => {
	const diff = Object.keys(obj1).reduce((result, key) => {
		if (!Object.prototype.hasOwnProperty.call(obj2, key)) {
			result.push(key);
		} else if (
			_.isEqual(obj1[key], obj2[key]) ||
			(obj1[key] === null && obj2[key] === '') ||
			(obj1[key] &&
				Array.isArray(obj1[key]) &&
				(obj1[key] as unknown[]).length === 0 &&
				obj2[key] === null)
		) {
			const resultKeyIndex = result.indexOf(key);
			if (resultKeyIndex >= 0) {
				result.splice(resultKeyIndex, 1);
			}
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
			/^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/,
		)
		? true
		: false;
};

export const validateURI = (uri: string) => {
	if (!uri || uri === '') return true;
	return uri
		.toLowerCase()
		.match(
			/(?:^|\s)((https?:\/\/)?(?:localhost|[\w-]+(?:\.[\w-]+)+)(:\d+)?(\/\S*)?)/,
		)
		? true
		: false;
};

/**
 * Extracts a user-facing message from a GraphQL/urql error.
 * Prefers the first GraphQL error message, then the error's message, then a fallback.
 */
export const getGraphQLErrorMessage = (
	error: unknown,
	fallback = 'Something went wrong',
): string => {
	if (!error || typeof error !== 'object') return fallback;
	const err = error as {
		message?: string;
		graphQLErrors?: Array<{ message?: string }>;
		networkError?: { message?: string };
	};
	if (Array.isArray(err.graphQLErrors) && err.graphQLErrors.length > 0) {
		const first = err.graphQLErrors[0];
		if (first?.message && typeof first.message === 'string') {
			return first.message;
		}
	}
	if (err.message && typeof err.message === 'string') return err.message;
	if (err.networkError?.message) return err.networkError.message;
	return fallback;
};
