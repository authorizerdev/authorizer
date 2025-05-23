import React, { MouseEventHandler, ReactNode } from 'react';
import { ButtonAppearance } from '../constants';
import styles from '../styles/default.css';

const StyledButton = ({
  style = {
    width: '100%',
  },
  type,
  appearance = ButtonAppearance.Default,
  disabled = false,
  onClick,
  children,
}: {
  type?: 'button' | 'submit' | 'reset' | undefined;
  style?: Record<string, string>;
  appearance?: ButtonAppearance;
  disabled?: boolean;
  onClick?: MouseEventHandler<HTMLSpanElement>;
  children: ReactNode;
}) => {
  return (
    <button
      className={styles['styled-button']}
      type={type}
      style={{
        width: style.width,
        backgroundColor: disabled
          ? 'var(--authorizer-primary-disabled-color)'
          : appearance === ButtonAppearance.Primary
          ? 'var(--authorizer-primary-color)'
          : 'var(--authorizer-white-color)',
        color:
          appearance === ButtonAppearance.Default
            ? 'var(--authorizer-text-color)'
            : 'var(--authorizer-white-color)',
        border: appearance === ButtonAppearance.Primary ? '0px' : '1px',
      }}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  );
};

export default StyledButton;
