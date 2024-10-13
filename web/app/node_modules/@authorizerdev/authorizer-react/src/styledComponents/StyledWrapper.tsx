import React, { ReactNode } from 'react';
import styles from '../styles/default.css';

const StyledWrapper = ({ children }: { children: ReactNode }) => {
  return <div className={styles['styled-wrapper']}>{children}</div>;
};

export default StyledWrapper;
