import { ReactNode } from 'react';
import '../styles/default.css';

const StyledSeparator = ({ children }: { children?: ReactNode }) => {
  return <div className="styled-separator">{children}</div>;
};

export default StyledSeparator;
