import React, { ReactNode } from 'react';
import styles from '../styles/default.css';

const StyledSeparator = ({ children }: { children?: ReactNode }) => {
  return <div className={styles['styled-separator']}>{children}</div>;
};

export default StyledSeparator;
