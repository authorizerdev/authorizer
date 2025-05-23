export const formatErrorMessage = (message: string): string => {
  return message.replace(`[GraphQL] `, '');
};

export const capitalizeFirstLetter = (data: string): string => {
  return data.charAt(0).toUpperCase() + data.slice(1);
};
