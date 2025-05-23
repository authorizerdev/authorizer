import React, { FC, useEffect, useState } from 'react';
import styles from '../styles/default.css';

import { ButtonAppearance, MessageType } from '../constants';
import { useAuthorizer } from '../contexts/AuthorizerContext';
import { StyledButton, StyledWrapper } from '../styledComponents';
import { formatErrorMessage } from '../utils/format';
import { Message } from './Message';
import { getSearchParams } from '../utils/url';
import PasswordStrengthIndicator from './PasswordStrengthIndicator';

type Props = {
  showOTPInput?: boolean;
  onReset?: (res: any) => void;
  phone_number?: string;
};

interface InputDataType {
  otp: string | null;
  password: string | null;
  confirmPassword: string | null;
}

export const AuthorizerResetPassword: FC<Props> = ({
  onReset,
  showOTPInput,
  phone_number,
}) => {
  const { token, redirect_uri } = getSearchParams();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<InputDataType>({
    otp: null,
    password: null,
    confirmPassword: null,
  });
  const [errorData, setErrorData] = useState<InputDataType>({
    otp: null,
    password: null,
    confirmPassword: null,
  });
  const { authorizerRef, config } = useAuthorizer();
  const [disableContinueButton, setDisableContinueButton] = useState(false);

  const onInputChange = async (field: string, value: string) => {
    setFormData({ ...formData, [field]: value });
  };

  const onSubmit = async (e: any) => {
    e.preventDefault();
    setLoading(true);
    try {
      const { data: res, errors } = await authorizerRef.resetPassword({
        token,
        otp: formData.otp || '',
        phone_number: phone_number || '',
        password: formData.password || '',
        confirm_password: formData.confirmPassword || '',
      });
      setLoading(false);
      if (errors && errors.length) {
        setError(formatErrorMessage(errors[0]?.message));
        return;
      }
      setError(``);
      if (onReset) {
        onReset(res);
      } else {
        window.location.href =
          redirect_uri || config.redirectURL || window.location.origin;
      }
    } catch (err) {
      setLoading(false);
      setError(formatErrorMessage((err as Error).message));
    }
  };

  const onErrorClose = () => {
    setError(``);
  };

  useEffect(() => {
    if (formData.password === '') {
      setErrorData({ ...errorData, password: 'Password is required' });
    } else {
      setErrorData({ ...errorData, password: null });
    }
  }, [formData.password]);

  useEffect(() => {
    if (formData.confirmPassword === '') {
      setErrorData({
        ...errorData,
        confirmPassword: 'Confirm password is required',
      });
    } else {
      setErrorData({ ...errorData, confirmPassword: null });
    }
  }, [formData.confirmPassword]);

  useEffect(() => {
    if (formData.password && formData.confirmPassword) {
      if (formData.confirmPassword !== formData.password) {
        setErrorData({
          ...errorData,
          password: `Password and confirm passwords don't match`,
          confirmPassword: `Password and confirm passwords don't match`,
        });
      } else {
        setErrorData({
          ...errorData,
          password: null,
          confirmPassword: null,
        });
      }
    }
  }, [formData.password, formData.confirmPassword]);

  return (
    <StyledWrapper>
      {error && (
        <Message type={MessageType.Error} text={error} onClose={onErrorClose} />
      )}
      <form onSubmit={onSubmit} name="authorizer-reset-password-form">
        {showOTPInput && (
          <div className={styles['styled-form-group']}>
            <label
              className={styles['form-input-label']}
              htmlFor="authorizer-verify-otp"
            >
              <span>* </span>OTP (One Time Password)
            </label>
            <input
              name="otp"
              id="authorizer-verify-otp"
              className={`${styles['form-input-field']} ${
                errorData.otp ? styles['input-error-content'] : null
              }`}
              placeholder="e.g.- AB123C"
              type="password"
              value={formData.otp || ''}
              onChange={(e) => onInputChange('otp', e.target.value)}
            />
            {errorData.otp && (
              <div className={styles['form-input-error']}>{errorData.otp}</div>
            )}
          </div>
        )}
        <div className={styles['styled-form-group']}>
          <label
            className={styles['form-input-label']}
            htmlFor="authorizer-reset-password"
          >
            <span>* </span>Password
          </label>
          <input
            name="password"
            id="authorizer-reset-password"
            className={`${styles['form-input-field']} ${
              errorData.password ? styles['input-error-content'] : null
            }`}
            placeholder="********"
            type="password"
            value={formData.password || ''}
            onChange={(e) => onInputChange('password', e.target.value)}
          />
          {errorData.password && (
            <div className={styles['form-input-error']}>
              {errorData.password}
            </div>
          )}
        </div>
        <div className={styles['styled-form-group']}>
          <label
            className={styles['form-input-label']}
            htmlFor="authorizer-reset-confirm-password"
          >
            <span>* </span>Confirm Password
          </label>
          <input
            name="confirmPassword"
            id="authorizer-reset-confirm-password"
            className={`${styles['form-input-field']} ${
              errorData.confirmPassword ? styles['input-error-content'] : null
            }`}
            placeholder="********"
            type="password"
            value={formData.confirmPassword || ''}
            onChange={(e) => onInputChange('confirmPassword', e.target.value)}
          />
          {errorData.confirmPassword && (
            <div className={styles['form-input-error']}>
              {errorData.confirmPassword}
            </div>
          )}
        </div>
        {config.is_strong_password_enabled && (
          <>
            <PasswordStrengthIndicator
              value={formData.password || ''}
              setDisableButton={setDisableContinueButton}
            />
            <br />
          </>
        )}
        <StyledButton
          type="submit"
          disabled={
            loading ||
            disableContinueButton ||
            !!errorData.password ||
            !!errorData.confirmPassword ||
            !formData.password ||
            !formData.confirmPassword
          }
          appearance={ButtonAppearance.Primary}
        >
          {loading ? `Processing ...` : `Continue`}
        </StyledButton>
      </form>
    </StyledWrapper>
  );
};
