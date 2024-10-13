import React, { FC, useEffect, useState } from 'react';
import isEmail from 'validator/es/lib/isEmail';

import styles from '../styles/default.css';
import { ButtonAppearance, MessageType } from '../constants';
import { useAuthorizer } from '../contexts/AuthorizerContext';
import { StyledButton } from '../styledComponents';
import { formatErrorMessage } from '../utils/format';
import { Message } from './Message';
import { MagicLinkLoginInput } from '@authorizerdev/authorizer-js';

interface InputDataType {
  email: string | null;
}

export const AuthorizerMagicLinkLogin: FC<{
  onMagicLinkLogin?: (data: any) => void;
  urlProps?: Record<string, any>;
  roles?: string[];
}> = ({ onMagicLinkLogin, urlProps, roles }) => {
  const [error, setError] = useState(``);
  const [loading, setLoading] = useState(false);
  const [successMessage, setSuccessMessage] = useState(``);
  const [formData, setFormData] = useState<InputDataType>({
    email: null,
  });
  const [errorData, setErrorData] = useState<InputDataType>({
    email: null,
  });
  const { authorizerRef } = useAuthorizer();

  const onInputChange = async (field: string, value: string) => {
    setFormData({ ...formData, [field]: value });
  };

  const onSubmit = async (e: any) => {
    e.preventDefault();
    try {
      setLoading(true);

      const data: MagicLinkLoginInput = {
        email: formData.email || '',
        state: urlProps?.state || '',
        redirect_uri: urlProps?.redirect_uri || '',
      };

      if (roles && roles.length) {
        data.roles = roles;
      }
      const { data: res, errors } = await authorizerRef.magicLinkLogin(data);
      setLoading(false);
      if (errors && errors.length) {
        setError(formatErrorMessage(errors[0]?.message));
        return;
      }

      if (res) {
        setError(``);
        setSuccessMessage(res.message || ``);

        if (onMagicLinkLogin) {
          onMagicLinkLogin(res);
        }
      }

      if (urlProps?.redirect_uri) {
        setTimeout(() => {
          window.location.replace(urlProps.redirect_uri);
        }, 3000);
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
    if (formData.email === '') {
      setErrorData({ ...errorData, email: 'Email is required' });
    } else if (formData.email && !isEmail(formData.email)) {
      setErrorData({ ...errorData, email: 'Please enter valid email' });
    } else {
      setErrorData({ ...errorData, email: null });
    }
  }, [formData.email]);

  if (successMessage) {
    return <Message type={MessageType.Success} text={successMessage} />;
  }

  return (
    <>
      {error && (
        <Message type={MessageType.Error} text={error} onClose={onErrorClose} />
      )}
      <form onSubmit={onSubmit} name="authorizer-magic-login-form">
        <div className={styles['styled-form-group']}>
          <label
            className={styles['form-input-label']}
            htmlFor="authorizer-magic-link-login-email"
          >
            <span>* </span>Email
          </label>
          <input
            name="email"
            id="authorizer-magic-link-login-email"
            className={`${styles['form-input-field']} ${
              errorData.email ? styles['input-error-content'] : null
            }`}
            placeholder="eg. foo@bar.com"
            type="email"
            value={formData.email || ''}
            onChange={(e) => onInputChange('email', e.target.value)}
          />
          {errorData.email && (
            <div className={styles['form-input-error']}>{errorData.email}</div>
          )}
        </div>
        <br />
        <StyledButton
          type="submit"
          disabled={loading || !!errorData.email || !formData.email}
          appearance={ButtonAppearance.Primary}
        >
          {loading ? `Processing ...` : `Send Email`}
        </StyledButton>
      </form>
    </>
  );
};
