import React, { FC, useState } from 'react';
import { StyledButton, StyledFlex, StyledSeparator } from '../styledComponents';
import { ButtonAppearance, Views } from '../constants';
import { AuthorizerVerifyOtp } from './AuthorizerVerifyOtp';

export const AuthorizerTOTPScanner: FC<{
  setView?: (v: Views) => void;
  onLogin?: (data: any) => void;
  email?: string;
  phone_number?: string;
  urlProps?: Record<string, any>;
  authenticator_scanner_image: string;
  authenticator_secret: string;
  authenticator_recovery_codes: string[];
}> = ({
  setView,
  onLogin,
  email,
  phone_number,
  authenticator_scanner_image,
  authenticator_secret,
  authenticator_recovery_codes,
  urlProps,
}) => {
  const [isOTPScreenVisisble, setIsOTPScreenVisisble] =
    useState<boolean>(false);

  const handleContinue = () => {
    setIsOTPScreenVisisble(true);
  };

  if (isOTPScreenVisisble) {
    return (
      <AuthorizerVerifyOtp
        {...{
          setView,
          onLogin,
          email,
          phone_number,
          urlProps,
        }}
        is_totp
      />
    );
  }

  return (
    <>
      <p style={{ margin: '10px 0px', fontWeight: 'bold' }}>
        Scan the QR code or enter the secret key into your authenticator app.
      </p>
      <StyledFlex justifyContent="center">
        <img
          src={`data:image/jpeg;base64,${authenticator_scanner_image}`}
          alt="scanner"
        />
      </StyledFlex>
      <p style={{ margin: '10px 0px' }}>
        If you are unable to scan the QR code, please enter the secret key
        manually.
      </p>
      <p style={{ margin: '10px 0px', fontWeight: 'bold' }}>
        {authenticator_secret}
      </p>
      <StyledSeparator />
      <p style={{ margin: '10px 0px' }}>
        If you lose access to your authenticator app, you can use the recovery
        codes below to regain access to your account. Please save these codes
        safely and do not share them with anyone.
      </p>
      <ul>
        {authenticator_recovery_codes.map((code, index) => {
          return <li key={index}>{code}</li>;
        })}
      </ul>
      <StyledButton
        type="button"
        appearance={ButtonAppearance.Primary}
        onClick={handleContinue}
      >
        Continue
      </StyledButton>
    </>
  );
};
