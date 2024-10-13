export const isValidOtp = (otp: string): boolean => {
  const re = /^([A-Z0-9]{6})$/;
  return re.test(String(otp.trim()));
};

export const hasSpecialChar = (char: string): boolean => {
  const re = /[`!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~]/;
  return re.test(char);
};

export const validatePassword = (
  value: string
): {
  score: number;
  strength: string;
  hasSixChar: boolean;
  hasLowerCase: boolean;
  hasUpperCase: boolean;
  hasNumericChar: boolean;
  hasSpecialChar: boolean;
  maxThirtySixChar: boolean;
  isValid: boolean;
} => {
  const res = {
    score: 0,
    strength: '',
    hasSixChar: false,
    hasLowerCase: false,
    hasUpperCase: false,
    hasNumericChar: false,
    hasSpecialChar: false,
    maxThirtySixChar: false,
  };

  if (value.length >= 6) {
    res.score = res.score + 1;
    res.hasSixChar = true;
  }

  if (value.length > 0 && value.length <= 36) {
    res.score = res.score + 1;
    res.maxThirtySixChar = true;
  }

  Array.from(value).forEach((char: any) => {
    if (char >= 'A' && char <= 'Z' && !res.hasUpperCase) {
      res.score = res.score + 1;
      res.hasUpperCase = true;
    } else if (char >= 'a' && char <= 'z' && !res.hasLowerCase) {
      res.score = res.score + 1;
      res.hasLowerCase = true;
    } else if (char >= '0' && char <= '9' && !res.hasNumericChar) {
      res.score = res.score + 1;
      res.hasNumericChar = true;
    } else if (hasSpecialChar(char) && !res.hasSpecialChar) {
      res.score = res.score + 1;
      res.hasSpecialChar = true;
    }
  });

  if (res.score <= 2) {
    res.strength = 'Weak';
  } else if (res.score <= 4) {
    res.strength = 'Good';
  } else if (res.score <= 5) {
    res.strength = 'Strong';
  } else {
    res.strength = 'Very Strong';
  }

  const isValid = Object.values(res).every((i) => Boolean(i));
  return { ...res, isValid };
};
