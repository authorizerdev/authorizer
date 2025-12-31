import { ReactNode } from 'react';
import '../styles/default.css';

const StyledFooter = ({ children }: { children: ReactNode }) => {
  return <div className="styled-footer">{children}</div>;
};

export default StyledFooter;
