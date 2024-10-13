import { AuthorizerConfig } from '../types';

export const getEmailPhoneLabels = (config: AuthorizerConfig): string => {
  const emailLabel = 'Email';
  const phoneLabel = 'Phone Number';
  if (
    config.is_basic_authentication_enabled &&
    config.is_mobile_basic_authentication_enabled
  ) {
    return `${emailLabel} / ${phoneLabel}`;
  } else if (config.is_basic_authentication_enabled) {
    return emailLabel;
  } else if (config.is_mobile_basic_authentication_enabled) {
    return phoneLabel;
  }
  return emailLabel;
};

export const getEmailPhonePlaceholder = (config: AuthorizerConfig): string => {
  const emailPlaceholder = 'hello@world.com';
  const phonePlaceholder = '+919999999999';
  const prefix = 'eg.';
  if (
    config.is_basic_authentication_enabled &&
    config.is_mobile_basic_authentication_enabled
  ) {
    return `${prefix} ${emailPlaceholder} / ${phonePlaceholder}`;
  } else if (config.is_basic_authentication_enabled) {
    return `${prefix} ${emailPlaceholder}`;
  } else if (config.is_mobile_basic_authentication_enabled) {
    return `${prefix} ${phonePlaceholder}`;
  }
  return emailPlaceholder;
};
