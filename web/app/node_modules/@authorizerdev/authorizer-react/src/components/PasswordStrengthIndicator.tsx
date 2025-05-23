import React from 'react';
import {
  StyledFlex,
  StyledPasswordStrengthWrapper,
  StyledPasswordStrength,
} from '../styledComponents';
import { validatePassword } from '../utils/validations';

interface PropTypes {
  value: string;
  setDisableButton: Function;
}

const PasswordStrengthIndicator = ({ value, setDisableButton }: PropTypes) => {
  const [
    {
      strength,
      score,
      hasSixChar,
      hasLowerCase,
      hasNumericChar,
      hasSpecialChar,
      hasUpperCase,
      maxThirtySixChar,
    },
    setValidations,
  ] = React.useState({ ...validatePassword(value || '') });

  React.useEffect(() => {
    const validationData = validatePassword(value || '');
    setValidations({ ...validationData });
    if (!validationData.isValid) {
      setDisableButton(true);
    } else {
      setDisableButton(false);
    }
  }, [value]);

  return (
    <div>
      <StyledPasswordStrengthWrapper>
        <StyledFlex alignItems="center" justifyContent="center" wrap="nowrap">
          <StyledPasswordStrength strength={score > 2 ? `weak` : `default`} />
          <StyledPasswordStrength strength={score > 3 ? `good` : `default`} />
          <StyledPasswordStrength strength={score > 4 ? `strong` : `default`} />
          <StyledPasswordStrength
            strength={score > 5 ? `veryStrong` : `default`}
          />
          {!!score && <div>{strength}</div>}
        </StyledFlex>
      </StyledPasswordStrengthWrapper>
      <p>
        <b>Criteria for a strong password:</b>
      </p>
      <StyledFlex flexDirection="column">
        <StyledFlex
          justifyContent="flex-start"
          alignItems="center"
          width="100%"
        >
          <input readOnly type="checkbox" checked={hasSixChar} />
          <div style={{ marginLeft: '5px' }}>At least 6 characters</div>
        </StyledFlex>
        <StyledFlex
          justifyContent="flex-start"
          alignItems="center"
          width="100%"
        >
          <input readOnly type="checkbox" checked={hasLowerCase} />
          <div style={{ marginLeft: '5px' }}>At least 1 lowercase letter</div>
        </StyledFlex>
        <StyledFlex
          justifyContent="flex-start"
          alignItems="center"
          width="100%"
        >
          <input readOnly type="checkbox" checked={hasUpperCase} />
          <div style={{ marginLeft: '5px' }}>At least 1 uppercase letter</div>
        </StyledFlex>
        <StyledFlex
          justifyContent="flex-start"
          alignItems="center"
          width="100%"
        >
          <input readOnly type="checkbox" checked={hasNumericChar} />
          <div style={{ marginLeft: '5px' }}>At least 1 numeric character</div>
        </StyledFlex>
        <StyledFlex
          justifyContent="flex-start"
          alignItems="center"
          width="100%"
        >
          <input readOnly type="checkbox" checked={hasSpecialChar} />
          <div style={{ marginLeft: '5px' }}>At least 1 special character</div>
        </StyledFlex>
        <StyledFlex
          justifyContent="flex-start"
          alignItems="center"
          width="100%"
        >
          <input readOnly type="checkbox" checked={maxThirtySixChar} />
          <div style={{ marginLeft: '5px' }}>Maximum 36 characters</div>
        </StyledFlex>
      </StyledFlex>
    </div>
  );
};

export default PasswordStrengthIndicator;
