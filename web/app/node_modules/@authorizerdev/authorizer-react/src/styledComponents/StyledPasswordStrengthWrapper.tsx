import { ReactNode } from 'react';
import '../styles/default.css';

const StyledPasswordStrengthWrapper = ({
  children,
}: {
  children: ReactNode;
}) => {
  return <div className="styled-password-strength-wrapper">{children}</div>;
};

export default StyledPasswordStrengthWrapper;
