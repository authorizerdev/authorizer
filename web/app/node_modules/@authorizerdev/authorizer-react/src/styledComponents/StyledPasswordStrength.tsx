import React, { ReactNode } from 'react';
import { passwordStrengthIndicatorOpacity } from '../constants';
import styles from '../styles/default.css';

const StyledPasswordStrength = ({
  strength = 'default',
  children,
}: {
  strength: string;
  children?: ReactNode;
}) => {
  return (
    <div
      className={styles['styled-password-strength']}
      style={{ opacity: passwordStrengthIndicatorOpacity[strength] }}
    >
      {children}
    </div>
  );
};

export default StyledPasswordStrength;
