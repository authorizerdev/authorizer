import _flatten from 'lodash/flatten';
import { validateEmail } from '.';

interface DataTypes {
	value: string;
	isInvalid: boolean;
}

const parseCSV = (file: File, delimiter: string): Promise<DataTypes[]> => {
	return new Promise((resolve) => {
		const reader = new FileReader();

		reader.onload = (e: ProgressEvent<FileReader>) => {
			const lines = (e.target?.result as string).split('\n');
			let result = lines.map((line: string) => line.split(delimiter));
			const flattened = _flatten(result);
			resolve(
				flattened.map((email: string) => {
					return {
						value: email.trim(),
						isInvalid: !validateEmail(email.trim()),
					};
				}),
			);
		};

		reader.readAsText(file);
	});
};

export default parseCSV;
