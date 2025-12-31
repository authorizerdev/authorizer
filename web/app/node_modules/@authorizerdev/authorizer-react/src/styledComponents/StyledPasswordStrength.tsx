import { ReactNode } from 'react';
import { passwordStrengthIndicatorOpacity } from '../constants';
import '../styles/default.css';

const StyledPasswordStrength = ({
  strength = 'default',
  children,
}: {
  strength: string;
  children?: ReactNode;
}) => {
  return (
    <div
      className="styled-password-strength"
      style={{ opacity: passwordStrengthIndicatorOpacity[strength] }}
    >
      {children}
    </div>
  );
};

export default StyledPasswordStrength;
