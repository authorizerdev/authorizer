import { ReactNode } from 'react';
import '../styles/default.css';

const StyledWrapper = ({ children }: { children: ReactNode }) => {
  return <div className="styled-wrapper">{children}</div>;
};

export default StyledWrapper;
