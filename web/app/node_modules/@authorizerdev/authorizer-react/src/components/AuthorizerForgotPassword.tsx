import React, { FC, useEffect, useState } from 'react';
import isEmail from 'validator/es/lib/isEmail';
import isMobilePhone from 'validator/es/lib/isMobilePhone';

import styles from '../styles/default.css';
import { ButtonAppearance, MessageType, Views } from '../constants';
import { useAuthorizer } from '../contexts/AuthorizerContext';
import { StyledButton, StyledFooter, StyledLink } from '../styledComponents';
import { formatErrorMessage } from '../utils/format';
import { Message } from './Message';
import { OtpDataType } from '../types';
import { AuthorizerResetPassword } from './AuthorizerResetPassword';
import { getEmailPhoneLabels, getEmailPhonePlaceholder } from '../utils/labels';

interface InputDataType {
  email_or_phone_number: string | null;
}

const initOtpData: OtpDataType = {
  is_screen_visible: false,
  email: '',
  phone_number: '',
};

export const AuthorizerForgotPassword: FC<{
  setView?: (v: Views) => void;
  onForgotPassword?: (data: any) => void;
  onPasswordReset?: () => void;
  urlProps?: Record<string, any>;
}> = ({ setView, onForgotPassword, onPasswordReset, urlProps }) => {
  const [error, setError] = useState(``);
  const [loading, setLoading] = useState(false);
  const [successMessage, setSuccessMessage] = useState(``);
  const [otpData, setOtpData] = useState<OtpDataType>({ ...initOtpData });
  const [formData, setFormData] = useState<InputDataType>({
    email_or_phone_number: null,
  });
  const [errorData, setErrorData] = useState<InputDataType>({
    email_or_phone_number: null,
  });
  const { authorizerRef, config } = useAuthorizer();

  const onInputChange = async (field: string, value: string) => {
    setFormData({ ...formData, [field]: value });
  };

  const onSubmit = async (e: any) => {
    e.preventDefault();
    try {
      setLoading(true);
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
      const { data: res, errors } = await authorizerRef.forgotPassword({
        email: email,
        phone_number: phone_number,
        state: urlProps?.state || '',
        redirect_uri:
          urlProps?.redirect_uri ||
          config.redirectURL ||
          window.location.origin,
      });
      setLoading(false);
      if (errors && errors.length) {
        setError(formatErrorMessage(errors[0]?.message));
        return;
      }
      if (res?.message) {
        setError(``);
        setSuccessMessage(res.message);
        if (res?.should_show_mobile_otp_screen) {
          setOtpData({
            ...otpData,
            is_screen_visible: true,
            email: email,
            phone_number: phone_number,
          });
          return;
        }
      }
      if (onForgotPassword) {
        onForgotPassword(res);
      }
    } catch (err) {
      setLoading(false);
      setError(formatErrorMessage((err as Error)?.message));
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
      formData.email_or_phone_number !== null &&
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

  if (successMessage) {
    return (
      <>
        <Message type={MessageType.Success} text={successMessage} />
        {otpData.is_screen_visible && (
          <AuthorizerResetPassword
            showOTPInput
            onReset={onPasswordReset}
            phone_number={otpData.phone_number}
          />
        )}
      </>
    );
  }

  return (
    <>
      {error && (
        <Message type={MessageType.Error} text={error} onClose={onErrorClose} />
      )}
      <p style={{ textAlign: 'center', margin: '10px 0px' }}>
        Please enter your {getEmailPhoneLabels(config)}.
        <br /> We will send you an email / otp to reset your password.
      </p>
      <br />
      <form onSubmit={onSubmit} name="authorizer-forgot-password-form">
        <div className={styles['styled-form-group']}>
          <label
            className={styles['form-input-label']}
            htmlFor="authorizer-forgot-password-email-or-phone-number"
          >
            <span>* </span>
            {getEmailPhoneLabels(config)}
          </label>
          <input
            name="email_or_phone_number"
            id="authorizer-forgot-password-email-or-phone-number"
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
        <br />
        <StyledButton
          type="submit"
          disabled={
            loading ||
            !!errorData.email_or_phone_number ||
            !formData.email_or_phone_number
          }
          appearance={ButtonAppearance.Primary}
        >
          {loading ? `Processing ...` : `Request Change`}
        </StyledButton>
      </form>
      {setView && (
        <StyledFooter>
          <div>
            Remember your password?{' '}
            <StyledLink onClick={() => setView(Views.Login)}>Log In</StyledLink>
          </div>
        </StyledFooter>
      )}
    </>
  );
};
