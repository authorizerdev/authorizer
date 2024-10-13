import React, { ReactNode } from 'react';
import styles from '../styles/default.css';

const StyledFlex = ({
  flexDirection = 'row',
  alignItems = 'center',
  justifyContent = 'center',
  wrap = 'wrap',
  width = 'inherit',
  children,
}: {
  flexDirection?: 'row' | 'row-reverse' | 'column' | 'column-reverse';
  alignItems?: string;
  justifyContent?: string;
  wrap?: 'nowrap' | 'wrap' | 'wrap-reverse';
  width?: string;
  children: ReactNode;
}) => {
  return (
    <div
      className={styles['styled-flex']}
      style={{
        flexDirection,
        alignItems,
        justifyContent,
        flexWrap: wrap,
        width: width,
      }}
    >
      {children}
    </div>
  );
};

export default StyledFlex;
