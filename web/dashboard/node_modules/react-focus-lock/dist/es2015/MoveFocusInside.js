import _extends from "@babel/runtime/helpers/esm/extends";
import React, { useEffect, useRef } from 'react';
import PropTypes from 'prop-types';
import { FOCUS_AUTO } from 'focus-lock/constants';
import { inlineProp } from './util';
import { mediumEffect } from './medium';
export var useFocusInside = function useFocusInside(observedRef) {
  useEffect(function () {
    var enabled = true;
    mediumEffect.useMedium(function (car) {
      var observed = observedRef && observedRef.current;
      if (enabled && observed) {
        if (!car.focusInside(observed)) {
          car.moveFocusInside(observed, null);
        }
      }
    });
    return function () {
      enabled = false;
    };
  }, [observedRef]);
};
function MoveFocusInside(_ref) {
  var _ref$disabled = _ref.disabled,
    isDisabled = _ref$disabled === void 0 ? false : _ref$disabled,
    className = _ref.className,
    children = _ref.children;
  var ref = useRef(null);
  useFocusInside(isDisabled ? undefined : ref);
  return /*#__PURE__*/React.createElement("div", _extends({}, inlineProp(FOCUS_AUTO, !isDisabled), {
    ref: ref,
    className: className
  }), children);
}
MoveFocusInside.propTypes = process.env.NODE_ENV !== "production" ? {
  children: PropTypes.node.isRequired,
  disabled: PropTypes.bool,
  className: PropTypes.string
} : {};
export default MoveFocusInside;