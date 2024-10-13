import React, { MouseEventHandler, ReactNode } from 'react';
import styles from '../styles/default.css';

const StyledLink = ({
  marginBottom = '0px',
  children,
  onClick,
}: {
  marginBottom?: string;
  children: ReactNode;
  onClick: MouseEventHandler<HTMLSpanElement>;
}) => {
  return (
    <span
      className={styles['styled-link']}
      style={{ marginBottom }}
      onClick={onClick}
    >
      {children}
    </span>
  );
};

export default StyledLink;
