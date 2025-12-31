import { FC, useEffect, useState } from 'react';
import { AuthToken, SignUpRequest } from '@authorizerdev/authorizer-js';
import isEmail from 'validator/es/lib/isEmail';
import isMobilePhone from 'validator/es/lib/isMobilePhone';

import '../styles/default.css';
import { ButtonAppearance, MessageType, Views } from '../constants';
import { useAuthorizer } from '../contexts/AuthorizerContext';
import { StyledButton, StyledFooter, StyledLink } from '../styledComponents';
import { formatErrorMessage } from '../utils/format';
import { Message } from './Message';
import PasswordStrengthIndicator from './PasswordStrengthIndicator';
import { OtpDataType } from '../types';
import { AuthorizerVerifyOtp } from './AuthorizerVerifyOtp';
import { getEmailPhoneLabels, getEmailPhonePlaceholder } from '../utils/labels';

type Field =
  | 'given_name'
  | 'family_name'
  | 'email_or_phone_number'
  | 'password'
  | 'confirmPassword';

type FieldOverride = {
  label: string;
  placeholder: string;
  hide?: boolean;
  notRequired?: boolean;
};

type InputDataType = {
  [K in Field]: string | null;
};

export type FormFieldsOverrides = {
  [K in Field]?: FieldOverride;
};

const initOtpData: OtpDataType = {
  is_screen_visible: false,
  email: '',
  phone_number: '',
};

export const AuthorizerSignup: FC<{
  setView?: (v: Views) => void;
  onSignup?: (data: AuthToken) => void;
  urlProps?: Record<string, any>;
  roles?: string[];
  fieldOverrides?: FormFieldsOverrides;
}> = ({ setView, onSignup, urlProps, roles, fieldOverrides }) => {
  const [error, setError] = useState(``);
  const [loading, setLoading] = useState(false);
  const [otpData, setOtpData] = useState<OtpDataType>({ ...initOtpData });
  const [successMessage, setSuccessMessage] = useState(``);
  const [formData, setFormData] = useState<InputDataType>({
    given_name: null,
    family_name: null,
    email_or_phone_number: null,
    password: null,
    confirmPassword: null,
  });
  const [errorData, setErrorData] = useState<InputDataType>({
    given_name: null,
    family_name: null,
    email_or_phone_number: null,
    password: null,
    confirmPassword: null,
  });
  const { authorizerRef, config, setAuthData } = useAuthorizer();
  const [disableSignupButton, setDisableSignupButton] = useState(false);

  const onInputChange = async (field: string, value: string) =>
    setFormData({ ...formData, [field]: value });

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
      const data: SignUpRequest = {
        email,
        phone_number,
        given_name: formData.given_name || '',
        family_name: formData.family_name || '',
        password: formData.password || '',
        confirm_password: formData.confirmPassword || '',
      };
      if (urlProps?.scope) {
        data.scope = urlProps.scope;
      }
      if (urlProps?.roles) {
        data.roles = urlProps.roles;
      }
      if (urlProps?.redirect_uri) {
        data.redirect_uri = urlProps.redirect_uri;
      }
      if (urlProps?.state) {
        data.state = urlProps.state;
      }
      if (roles && roles.length) {
        data.roles = roles;
      }
      const { data: res, errors } = await authorizerRef.signup(data);
      if (errors && errors.length) {
        setError(formatErrorMessage(errors[0]?.message));
        setLoading(false);
        return;
      }
      if (
        res &&
        (res?.should_show_email_otp_screen ||
          res?.should_show_mobile_otp_screen)
      ) {
        setOtpData({
          is_screen_visible: true,
          email: data.email || ``,
          phone_number: data.phone_number || ``,
        });
        return;
      }
      if (res) {
        setError(``);
        if (res.access_token) {
          setError(``);
          setAuthData({
            user: res.user || null,
            token: res,
            config,
            loading: false,
          });
        } else {
          setLoading(false);
          setSuccessMessage(res.message || ``);
        }

        if (onSignup) {
          onSignup(res);
        }
      }
    } catch (err) {
      setLoading(false);
      setError(formatErrorMessage((err as Error).message));
    }
  };

  const onErrorClose = () => setError(``);

  useEffect(() => {
    if (
      fieldOverrides?.given_name?.notRequired ||
      fieldOverrides?.given_name?.hide
    ) {
      return;
    }
    if ((formData.given_name || '').trim() === '') {
      setErrorData({ ...errorData, given_name: 'First Name is required' });
    } else {
      setErrorData({ ...errorData, given_name: null });
    }
  }, [formData.given_name]);

  useEffect(() => {
    if (
      fieldOverrides?.family_name?.notRequired ||
      fieldOverrides?.family_name?.hide
    ) {
      return;
    }
    if ((formData.family_name || '').trim() === '') {
      setErrorData({ ...errorData, family_name: 'Last Name is required' });
    } else {
      setErrorData({ ...errorData, family_name: null });
    }
  }, [formData.family_name]);

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

  if (otpData.is_screen_visible) {
    return (
      <>
        {successMessage && (
          <Message type={MessageType.Success} text={successMessage} />
        )}
        <AuthorizerVerifyOtp
          {...{
            setView,
            onLogin: onSignup,
            email: otpData.email || ``,
            phone_number: otpData.phone_number || ``,
            is_totp: otpData.is_totp || false,
          }}
          urlProps={urlProps}
        />
      </>
    );
  }

  const renderField = (
    key: Field,
    label: string,
    placeholder: string,
    type?: 'text' | 'password'
  ) => {
    const fieldOverride = fieldOverrides?.[key];
    if (fieldOverride?.hide) {
      return null;
    }
    return (
      <div className="styled-form-group">
        <label
          className="form-input-label"
          htmlFor={`authorizer-sign-up-${key}`}
        >
          {!fieldOverride?.notRequired && <span>* </span>}
          {fieldOverride?.label ?? label}
        </label>
        <input
          name={key}
          id={`authorizer-sign-up-${key}`}
          className={`form-input-field ${
            errorData[key] ? 'input-error-content' : ''
          }`}
          placeholder={fieldOverride?.placeholder ?? placeholder}
          type={type}
          value={formData[key] || ''}
          onChange={(e) => onInputChange(key, e.target.value)}
        />
        {errorData[key] && (
          <div className="form-input-error">{errorData[key]}</div>
        )}
      </div>
    );
  };

  const shouldFieldBlockSubmit = (key: Field) => {
    if (
      (formData[key] ||
        fieldOverrides?.[key]?.notRequired ||
        fieldOverrides?.[key]?.hide) &&
      !errorData[key]
    ) {
      return false;
    }
    return true;
  };

  return (
    <>
      {error && (
        <Message type={MessageType.Error} text={error} onClose={onErrorClose} />
      )}
      {successMessage && (
        <Message type={MessageType.Success} text={successMessage} />
      )}
      {(config.is_basic_authentication_enabled ||
        config.is_mobile_basic_authentication_enabled) &&
        !config.is_magic_link_login_enabled && (
          <>
            <form onSubmit={onSubmit} name="authorizer-sign-up-form">
              {renderField('given_name', 'First Name', 'eg. John', 'text')}
              {renderField('family_name', 'Last Name', 'eg. Doe', 'text')}
              {renderField(
                'email_or_phone_number',
                getEmailPhoneLabels(config),
                getEmailPhonePlaceholder(config)
              )}
              {renderField('password', 'Password', '********', 'password')}
              {renderField(
                'confirmPassword',
                'Confirm Password',
                '********',
                'password'
              )}
              {config.is_strong_password_enabled && (
                <>
                  <PasswordStrengthIndicator
                    value={formData.password || ''}
                    setDisableButton={setDisableSignupButton}
                  />
                  <br />
                </>
              )}
              <br />
              <StyledButton
                type="submit"
                disabled={
                  loading ||
                  disableSignupButton ||
                  shouldFieldBlockSubmit('given_name') ||
                  shouldFieldBlockSubmit('family_name') ||
                  !!errorData.email_or_phone_number ||
                  !!errorData.password ||
                  !!errorData.confirmPassword ||
                  !formData.email_or_phone_number ||
                  !formData.password ||
                  !formData.confirmPassword
                }
                appearance={ButtonAppearance.Primary}
              >
                {loading ? `Processing ...` : `Sign Up`}
              </StyledButton>
            </form>
            {setView && (
              <StyledFooter>
                <div>
                  Already have an account?{' '}
                  <StyledLink onClick={() => setView(Views.Login)}>
                    Log In
                  </StyledLink>
                </div>
              </StyledFooter>
            )}
          </>
        )}
    </>
  );
};
