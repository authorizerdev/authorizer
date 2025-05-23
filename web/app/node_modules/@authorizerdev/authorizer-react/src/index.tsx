import {
  AuthorizerProvider,
  useAuthorizer,
} from './contexts/AuthorizerContext';
import { AuthorizerSignup } from './components/AuthorizerSignup';
import { AuthorizerBasicAuthLogin } from './components/AuthorizerBasicAuthLogin';
import { AuthorizerMagicLinkLogin } from './components/AuthorizerMagicLinkLogin';
import { AuthorizerForgotPassword } from './components/AuthorizerForgotPassword';
import { AuthorizerSocialLogin } from './components/AuthorizerSocialLogin';
import { AuthorizerResetPassword } from './components/AuthorizerResetPassword';
import { AuthorizerVerifyOtp } from './components/AuthorizerVerifyOtp';
import { AuthorizerRoot as Authorizer } from './components/AuthorizerRoot';
import { AuthorizerTOTPScanner } from './components/AuthorizerTOTPScanner';

export {
  useAuthorizer,
  Authorizer,
  AuthorizerProvider,
  AuthorizerSignup,
  AuthorizerBasicAuthLogin,
  AuthorizerMagicLinkLogin,
  AuthorizerForgotPassword,
  AuthorizerSocialLogin,
  AuthorizerResetPassword,
  AuthorizerVerifyOtp,
  AuthorizerTOTPScanner,
};
