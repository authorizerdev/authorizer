export const hasAdminSecret = () => {
	return (<any>window)['__authorizer__'].isOnboardingCompleted === true;
};

export const capitalizeFirstLetter = (data: string): string =>
	data.charAt(0).toUpperCase() + data.slice(1);
