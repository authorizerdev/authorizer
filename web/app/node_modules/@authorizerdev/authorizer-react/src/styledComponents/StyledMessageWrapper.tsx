import React, { ReactNode } from 'react';
import { MessageType } from '../constants';
import styles from '../styles/default.css';

const getBackgroundColor = (type: MessageType): string => {
  switch (type) {
    case MessageType.Error:
      return 'var(--authorizer-danger-color)';
    case MessageType.Success:
      return 'var(--authorizer-success-color)';
    case MessageType.Info:
      return 'var(--authorizer-slate-color)';
    default:
      return 'var(--authorizer-success-color)';
  }
};

const StyledMessageWrapper = ({
  type = MessageType.Success,
  styles: extraStyles = {},
  children,
}: {
  type: MessageType;
  children: ReactNode;
  styles?: Record<string, string>;
}) => {
  return (
    <div
      className={styles['styled-message-wrapper']}
      style={{
        backgroundColor: getBackgroundColor(type),
        ...extraStyles,
      }}
    >
      {children}
    </div>
  );
};

export default StyledMessageWrapper;
