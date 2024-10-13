import React, { FC, useEffect, useState } from 'react';
import { AuthToken, LoginInput } from '@authorizerdev/authorizer-js';
import isEmail from 'validator/es/lib/isEmail';
import isMobilePhone from 'validator/es/lib/isMobilePhone';

import styles from '../styles/default.css';
import { ButtonAppearance, MessageType, Views } from '../constants';
import { useAuthorizer } from '../contexts/AuthorizerContext';
import { StyledButton, StyledFooter, StyledLink } from '../styledComponents';
import { Message } from './Message';
import { AuthorizerVerifyOtp } from './AuthorizerVerifyOtp';
import { OtpDataType, TotpDataType } from '../types';
import { AuthorizerTOTPScanner } from './AuthorizerTOTPScanner';
import { getEmailPhoneLabels, getEmailPhonePlaceholder } from '../utils/labels';

const initOtpData: OtpDataType = {
  is_screen_visible: false,
  email: '',
  phone_number: '',
};

const initTotpData: TotpDataType = {
  is_screen_visible: false,
  email: '',
  phone_number: '',
  authenticator_scanner_image: '',
  authenticator_secret: '',
  authenticator_recovery_codes: [],
};

interface InputDataType {
  email_or_phone_number: string | null;
  password: string | null;
}

export const AuthorizerBasicAuthLogin: FC<{
  setView?: (v: Views) => void;
  onLogin?: (data: AuthToken | void) => void;
  urlProps?: Record<string, any>;
  roles?: string[];
}> = ({ setView, onLogin, urlProps, roles }) => {
  const [error, setError] = useState(``);
  const [loading, setLoading] = useState(false);
  const [otpData, setOtpData] = useState<OtpDataType>({ ...initOtpData });
  const [totpData, setTotpData] = useState<TotpDataType>({ ...initTotpData });
  const [formData, setFormData] = useState<InputDataType>({
    email_or_phone_number: null,
    password: null,
  });
  const [errorData, setErrorData] = useState<InputDataType>({
    email_or_phone_number: null,
    password: null,
  });
  const { setAuthData, config, authorizerRef } = useAuthorizer();

  const onInputChange = async (field: string, value: string) => {
    setFormData({ ...formData, [field]: value });
  };

  const onSubmit = async (e: any) => {
    e.preventDefault();
    setLoading(true);
    try {
      let email: string = '';
      let phone_number: string = '';
      if (formData.email_or_phone_number) {
        if (isEmail(formData.email_or_phone_number)) {
          email = formData.email_or_phone_number;
        } else if (isMobilePhone(formData.email_or_phone_number)) {
          phone_number = formData.email_or_phone_number;
        }
      }
      if (!email && !phone_number) {
        setErrorData({
          ...errorData,
          email_or_phone_number: 'Invalid email or phone number',
        });
        setLoading(false);
        return;
      }
      const data: LoginInput = {
        email: email,
        phone_number: phone_number,
        password: formData.password || '',
      };
      if (urlProps?.scope) {
        data.scope = urlProps.scope;
      }
      if (urlProps?.state) {
        data.state = urlProps.state;
      }

      if (roles && roles.length) {
        data.roles = roles;
      }

      const { data: res, errors } = await authorizerRef.login(data);
      if (errors && errors.length) {
        setError(errors[0].message);
        setLoading(false);
        return;
      }
      // if totp is enabled for the first time show totp screen with scanner
      if (
        res &&
        res.should_show_totp_screen &&
        res.authenticator_scanner_image &&
        res.authenticator_secret &&
        res.authenticator_recovery_codes
      ) {
        setTotpData({
          is_screen_visible: true,
          email: data.email || ``,
          phone_number: data.phone_number || ``,
          authenticator_scanner_image: res.authenticator_scanner_image,
          authenticator_secret: res.authenticator_secret,
          authenticator_recovery_codes: res.authenticator_recovery_codes,
        });
        return;
      }
      if (
        res &&
        (res?.should_show_email_otp_screen ||
          res?.should_show_mobile_otp_screen ||
          res?.should_show_totp_screen)
      ) {
        setOtpData({
          is_screen_visible: true,
          email: data.email || ``,
          phone_number: data.phone_number || ``,
          is_totp: res?.should_show_totp_screen || false,
        });
        return;
      }

      if (res) {
        setError(``);
        setAuthData({
          user: res.user || null,
          token: {
            access_token: res.access_token,
            expires_in: res.expires_in,
            refresh_token: res.refresh_token,
            id_token: res.id_token,
          },
          config,
          loading: false,
        });
      }

      if (onLogin) {
        onLogin(res);
      }
    } catch (err) {
      setLoading(false);
      setError((err as Error).message);
    }
  };

  const onErrorClose = () => {
    setError(``);
  };

  useEffect(() => {
    if (formData.email_or_phone_number === '') {
      setErrorData({
        ...errorData,
        email_or_phone_number: 'Email OR Phone Number is required',
      });
    } else if (
      !isEmail(formData.email_or_phone_number || '') &&
      !isMobilePhone(formData.email_or_phone_number || '')
    ) {
      setErrorData({
        ...errorData,
        email_or_phone_number: 'Invalid Email OR Phone Number',
      });
    } else {
      setErrorData({ ...errorData, email_or_phone_number: null });
    }
  }, [formData.email_or_phone_number]);

  useEffect(() => {
    if (formData.password === '') {
      setErrorData({ ...errorData, password: 'Password is required' });
    } else {
      setErrorData({ ...errorData, password: null });
    }
  }, [formData.password]);

  if (totpData.is_screen_visible) {
    return (
      <AuthorizerTOTPScanner
        {...{
          setView,
          onLogin,
          email: totpData.email || ``,
          phone_number: totpData.phone_number || ``,
          authenticator_scanner_image: totpData.authenticator_scanner_image,
          authenticator_secret: totpData.authenticator_secret,
          authenticator_recovery_codes:
            totpData.authenticator_recovery_codes || [],
        }}
        urlProps={urlProps}
      />
    );
  }

  if (otpData.is_screen_visible) {
    return (
      <AuthorizerVerifyOtp
        {...{
          setView,
          onLogin,
          email: otpData.email || ``,
          phone_number: otpData.phone_number || ``,
          is_totp: otpData.is_totp || false,
        }}
        urlProps={urlProps}
      />
    );
  }

  return (
    <>
      {error && (
        <Message type={MessageType.Error} text={error} onClose={onErrorClose} />
      )}
      <>
        <form onSubmit={onSubmit} name="authorizer-login-form">
          <div className={styles['styled-form-group']}>
            <label
              className={styles['form-input-label']}
              htmlFor="authorizer-login-email"
            >
              <span>* </span>
              {getEmailPhoneLabels(config)}
            </label>
            <input
              name="email_or_phone_number"
              id="authorizer-login-email-or-phone-number"
              className={`${styles['form-input-field']} ${
                errorData.email_or_phone_number
                  ? styles['input-error-content']
                  : null
              }`}
              placeholder={getEmailPhonePlaceholder(config)}
              type="text"
              value={formData.email_or_phone_number || ''}
              onChange={e =>
                onInputChange('email_or_phone_number', e.target.value)
              }
            />
            {errorData.email_or_phone_number && (
              <div className={styles['form-input-error']}>
                {errorData.email_or_phone_number}
              </div>
            )}
          </div>
          <div className={styles['styled-form-group']}>
            <label
              className={styles['form-input-label']}
              htmlFor="authorizer-login-password"
            >
              <span>* </span>Password
            </label>
            <input
              name="password"
              id="authorizer-login-password"
              className={`${styles['form-input-field']} ${
                errorData.password ? styles['input-error-content'] : null
              }`}
              placeholder="********"
              type="password"
              value={formData.password || ''}
              onChange={e => onInputChange('password', e.target.value)}
            />
            {errorData.password && (
              <div className={styles['form-input-error']}>
                {errorData.password}
              </div>
            )}
          </div>
          <br />
          <StyledButton
            type="submit"
            disabled={
              !!errorData.email_or_phone_number ||
              !!errorData.password ||
              !formData.email_or_phone_number ||
              !formData.password ||
              loading
            }
            appearance={ButtonAppearance.Primary}
          >
            {loading ? `Processing ...` : `Log In`}
          </StyledButton>
        </form>

        {setView && (
          <StyledFooter>
            <StyledLink
              onClick={() => setView(Views.ForgotPassword)}
              marginBottom="10px"
            >
              Forgot Password?
            </StyledLink>

            {config.is_sign_up_enabled && (
              <div>
                Don't have an account?{' '}
                <StyledLink onClick={() => setView(Views.Signup)}>
                  Sign Up
                </StyledLink>
              </div>
            )}
          </StyledFooter>
        )}
      </>
    </>
  );
};
