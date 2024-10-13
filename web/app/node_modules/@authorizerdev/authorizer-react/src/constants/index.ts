export enum Views {
  Login,
  Signup,
  ForgotPassword,
}

export enum ButtonAppearance {
  Primary,
  Default,
}

export enum MessageType {
  Error,
  Success,
  Info,
}

export enum AuthorizerProviderActionType {
  SET_USER = 'SET_USER',
  SET_TOKEN = 'SET_TOKEN',
  SET_LOADING = 'SET_LOADING',
  SET_AUTH_DATA = 'SET_AUTH_DATA',
  SET_CONFIG = 'SET_CONFIG',
}

// TODO use based on theme primary color
export const passwordStrengthIndicatorOpacity: Record<string, number> = {
  default: 0.15,
  weak: 0.4,
  good: 0.6,
  strong: 0.8,
  veryStrong: 1,
};
