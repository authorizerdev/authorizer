import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

// isFgaNotEnabledError detects the backend error returned by every FGA
// resolver when the server is started without an FGA store (--fga-store).
// The backend message is "fine-grained authorization is not enabled".
export function isFgaNotEnabledError(error?: { message?: string } | null) {
	if (!error?.message) {
		return false;
	}
	return error.message
		.toLowerCase()
		.includes('fine-grained authorization is not enabled');
}
