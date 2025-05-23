import React, { FC, useState } from 'react';
import { AuthToken } from '@authorizerdev/authorizer-js';

import { AuthorizerBasicAuthLogin } from './AuthorizerBasicAuthLogin';
import { useAuthorizer } from '../contexts/AuthorizerContext';
import { StyledWrapper } from '../styledComponents';
import { Views } from '../constants';
import { AuthorizerSignup } from './AuthorizerSignup';
import { AuthorizerForgotPassword } from './AuthorizerForgotPassword';
import { AuthorizerSocialLogin } from './AuthorizerSocialLogin';
import { AuthorizerMagicLinkLogin } from './AuthorizerMagicLinkLogin';
import { createRandomString } from '../utils/common';
import { hasWindow } from '../utils/window';

export const AuthorizerRoot: FC<{
  onLogin?: (data: AuthToken | void) => void;
  onSignup?: (data: AuthToken | void) => void;
  onMagicLinkLogin?: (data: any) => void;
  onForgotPassword?: (data: any) => void;
  onPasswordReset?: () => void;
  roles?: string[];
}> = ({
  onLogin,
  onSignup,
  onMagicLinkLogin,
  onForgotPassword,
  onPasswordReset,
  roles,
}) => {
  const [view, setView] = useState(Views.Login);
  const { config } = useAuthorizer();
  const searchParams = new URLSearchParams(
    hasWindow() ? window.location.search : ``
  );
  const state = searchParams.get('state') || createRandomString();
  const scope = searchParams.get('scope')
    ? searchParams
        .get('scope')
        ?.toString()
        .split(' ')
    : ['openid', 'profile', 'email'];

  const urlProps: Record<string, any> = {
    state,
    scope,
  };

  const redirectURL =
    searchParams.get('redirect_uri') || searchParams.get('redirectURL');
  if (redirectURL) {
    urlProps.redirectURL = redirectURL;
  } else {
    urlProps.redirectURL = hasWindow() ? window.location.origin : redirectURL;
  }

  urlProps.redirect_uri = urlProps.redirectURL;
  return (
    <StyledWrapper>
      <AuthorizerSocialLogin urlProps={urlProps} roles={roles} />
      {view === Views.Login &&
        (config.is_basic_authentication_enabled ||
          config.is_mobile_basic_authentication_enabled) &&
        !config.is_magic_link_login_enabled && (
          <AuthorizerBasicAuthLogin
            setView={setView}
            onLogin={onLogin}
            urlProps={urlProps}
            roles={roles}
          />
        )}

      {view === Views.Signup &&
        (config.is_basic_authentication_enabled ||
          config.is_mobile_basic_authentication_enabled) &&
        !config.is_magic_link_login_enabled &&
        config.is_sign_up_enabled && (
          <AuthorizerSignup
            setView={setView}
            onSignup={onSignup}
            urlProps={urlProps}
            roles={roles}
          />
        )}

      {view === Views.Login && config.is_magic_link_login_enabled && (
        <AuthorizerMagicLinkLogin
          onMagicLinkLogin={onMagicLinkLogin}
          urlProps={urlProps}
          roles={roles}
        />
      )}

      {view === Views.ForgotPassword && (
        <AuthorizerForgotPassword
          setView={setView}
          onForgotPassword={onForgotPassword}
          onPasswordReset={onPasswordReset}
          urlProps={urlProps}
        />
      )}
    </StyledWrapper>
  );
};
