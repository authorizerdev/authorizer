import React, { FC } from 'react';
import { MessageType } from '../constants';
import { IconClose } from '../icons/close';
import { StyledMessageWrapper, StyledFlex } from '../styledComponents';
import { capitalizeFirstLetter } from '../utils/format';

type Props = {
  type: MessageType;
  text: string;
  onClose?: () => void;
  extraStyles?: Record<string, string>;
};

export const Message: FC<Props> = ({ type, text, extraStyles, onClose }) => {
  if (text.trim()) {
    return (
      <StyledMessageWrapper type={type} styles={extraStyles}>
        <StyledFlex alignItems="center" justifyContent="space-between">
          <div style={{ flex: 1 }}>{capitalizeFirstLetter(text)}</div>
          {onClose && (
            <span style={{ cursor: 'pointer' }} onClick={onClose}>
              <IconClose />
            </span>
          )}
        </StyledFlex>
      </StyledMessageWrapper>
    );
  }

  return null;
};
