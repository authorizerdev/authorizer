import _flatten from "lodash/flatten";
import { validateEmail } from ".";

interface dataTypes {
  value: string;
  isInvalid: boolean;
}

const parseCSV = (file: File, delimiter: string): Promise<dataTypes[]> => {
  return new Promise((resolve) => {
    const reader = new FileReader();

    // When the FileReader has loaded the file...
    reader.onload = (e: any) => {
      // Split the result to an array of lines
      const lines = e.target.result.split("\n");
      // Split the lines themselves by the specified
      // delimiter, such as a comma
      let result = lines.map((line: string) => line.split(delimiter));
      // As the FileReader reads asynchronously,
      // we can't just return the result; instead,
      // we're passing it to a callback function
      result = _flatten(result);
      resolve(
        result.map((email: string) => {
          return {
            value: email.trim(),
            isInvalid: !validateEmail(email.trim()),
          };
        })
      );
    };

    // Read the file content as a single string
    reader.readAsText(file);
  });
};

export default parseCSV;
