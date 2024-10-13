import { useStyleConfig, omitThemingProps, forwardRef, chakra } from '@chakra-ui/system';
import { focus, ariaAttr, callAllHandlers, __DEV__, cx } from '@chakra-ui/utils';
import { createContext, mergeRefs, getValidChildren } from '@chakra-ui/react-utils';
import * as React from 'react';
import { createDescendantContext } from '@chakra-ui/descendant';
import { useId, useControllableState } from '@chakra-ui/hooks';

function _extends() {
  _extends = Object.assign || function (target) {
    for (var i = 1; i < arguments.length; i++) {
      var source = arguments[i];

      for (var key in source) {
        if (Object.prototype.hasOwnProperty.call(source, key)) {
          target[key] = source[key];
        }
      }
    }

    return target;
  };

  return _extends.apply(this, arguments);
}

function _objectWithoutPropertiesLoose(source, excluded) {
  if (source == null) return {};
  var target = {};
  var sourceKeys = Object.keys(source);
  var key, i;

  for (i = 0; i < sourceKeys.length; i++) {
    key = sourceKeys[i];
    if (excluded.indexOf(key) >= 0) continue;
    target[key] = source[key];
  }

  return target;
}

var _excluded$1 = ["index"];
/* -------------------------------------------------------------------------------------------------
 * Create context to track descendants and their indices
 * -----------------------------------------------------------------------------------------------*/

var _createDescendantCont = createDescendantContext(),
    PinInputDescendantsProvider = _createDescendantCont[0],
    usePinInputDescendantsContext = _createDescendantCont[1],
    usePinInputDescendants = _createDescendantCont[2],
    usePinInputDescendant = _createDescendantCont[3];

var _createContext = createContext({
  name: "PinInputContext",
  errorMessage: "usePinInputContext: `context` is undefined. Seems you forgot to all pin input fields within `<PinInput />`"
}),
    PinInputProvider = _createContext[0],
    usePinInputContext = _createContext[1];

var toArray = function toArray(value) {
  return value == null ? void 0 : value.split("");
};

function validate(value, type) {
  var NUMERIC_REGEX = /^[0-9]+$/;
  var ALPHA_NUMERIC_REGEX = /^[a-zA-Z0-9]+$/i;
  var regex = type === "alphanumeric" ? ALPHA_NUMERIC_REGEX : NUMERIC_REGEX;
  return regex.test(value);
}
/* -------------------------------------------------------------------------------------------------
 * usePinInput - handles the general pin input logic
 * -----------------------------------------------------------------------------------------------*/

/**
 * @internal
 */


function usePinInput(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      autoFocus = _props.autoFocus,
      value = _props.value,
      defaultValue = _props.defaultValue,
      _onChange = _props.onChange,
      onComplete = _props.onComplete,
      _props$placeholder = _props.placeholder,
      placeholder = _props$placeholder === void 0 ? "○" : _props$placeholder,
      _props$manageFocus = _props.manageFocus,
      manageFocus = _props$manageFocus === void 0 ? true : _props$manageFocus,
      _props$otp = _props.otp,
      otp = _props$otp === void 0 ? false : _props$otp,
      idProp = _props.id,
      isDisabled = _props.isDisabled,
      isInvalid = _props.isInvalid,
      _props$type = _props.type,
      type = _props$type === void 0 ? "number" : _props$type,
      mask = _props.mask;
  var uuid = useId();
  var id = idProp != null ? idProp : "pin-input-" + uuid;
  var descendants = usePinInputDescendants();

  var _React$useState = React.useState(true),
      moveFocus = _React$useState[0],
      setMoveFocus = _React$useState[1];

  var _React$useState2 = React.useState(-1),
      focusedIndex = _React$useState2[0],
      setFocusedIndex = _React$useState2[1];

  var _useControllableState = useControllableState({
    defaultValue: toArray(defaultValue) || [],
    value: toArray(value),
    onChange: function onChange(values) {
      return _onChange == null ? void 0 : _onChange(values.join(""));
    }
  }),
      values = _useControllableState[0],
      setValues = _useControllableState[1];

  React.useEffect(function () {
    if (autoFocus) {
      var first = descendants.first();
      if (first) focus(first.node, {
        nextTick: true
      });
    } // We don't want to listen for updates to `autoFocus` since it only runs initially
    // eslint-disable-next-line

  }, [descendants]);
  var focusNext = React.useCallback(function (index) {
    if (!moveFocus || !manageFocus) return;
    var next = descendants.next(index, false);
    if (next) focus(next.node, {
      nextTick: true
    });
  }, [descendants, moveFocus, manageFocus]);
  var setValue = React.useCallback(function (value, index) {
    var nextValues = [].concat(values);
    nextValues[index] = value;
    setValues(nextValues);
    var isComplete = value !== "" && nextValues.length === descendants.count() && nextValues.every(function (inputValue) {
      return inputValue != null && inputValue !== "";
    });

    if (isComplete) {
      onComplete == null ? void 0 : onComplete(nextValues.join(""));
    } else {
      focusNext(index);
    }
  }, [values, setValues, focusNext, onComplete, descendants]);
  var clear = React.useCallback(function () {
    var values = Array(descendants.count()).fill("");
    setValues(values);
    var first = descendants.first();
    if (first) focus(first.node);
  }, [descendants, setValues]);
  var getNextValue = React.useCallback(function (value, eventValue) {
    var nextValue = eventValue;

    if ((value == null ? void 0 : value.length) > 0) {
      if (value[0] === eventValue.charAt(0)) {
        nextValue = eventValue.charAt(1);
      } else if (value[0] === eventValue.charAt(1)) {
        nextValue = eventValue.charAt(0);
      }
    }

    return nextValue;
  }, []);
  var getInputProps = React.useCallback(function (props) {
    var index = props.index,
        rest = _objectWithoutPropertiesLoose(props, _excluded$1);
    /**
     * Improved from: https://github.com/uber/baseweb/blob/master/src/pin-code/pin-code.js
     */


    var onChange = function onChange(event) {
      var eventValue = event.target.value;
      var currentValue = values[index];
      var nextValue = getNextValue(currentValue, eventValue); // if the value was removed using backspace

      if (nextValue === "") {
        setValue("", index);
        return;
      } // in the case of an autocomplete or copy and paste


      if (eventValue.length > 2) {
        // see if we can use the string to fill out our values
        if (validate(eventValue, type)) {
          // Ensure the value matches the number of inputs
          var _nextValue = eventValue.split("").filter(function (_, index) {
            return index < descendants.count();
          });

          setValues(_nextValue); // if pasting fills the entire input fields, trigger `onComplete`

          if (_nextValue.length === descendants.count()) {
            onComplete == null ? void 0 : onComplete(_nextValue.join(""));
          }
        }
      } else {
        // only set if the new value is a number
        if (validate(nextValue, type)) {
          setValue(nextValue, index);
        }

        setMoveFocus(true);
      }
    };

    var onKeyDown = function onKeyDown(event) {
      if (event.key === "Backspace" && manageFocus) {
        if (event.target.value === "") {
          var prevInput = descendants.prev(index, false);

          if (prevInput) {
            setValue("", index - 1);
            focus(prevInput.node);
            setMoveFocus(true);
          }
        } else {
          setMoveFocus(false);
        }
      }
    };

    var onFocus = function onFocus() {
      setFocusedIndex(index);
    };

    var onBlur = function onBlur() {
      setFocusedIndex(-1);
    };

    var hasFocus = focusedIndex === index;
    var inputType = type === "number" ? "tel" : "text";
    return _extends({
      "aria-label": "Please enter your pin code",
      inputMode: type === "number" ? "numeric" : "text",
      type: mask ? "password" : inputType
    }, rest, {
      id: id + "-" + index,
      disabled: isDisabled,
      "aria-invalid": ariaAttr(isInvalid),
      onChange: callAllHandlers(rest.onChange, onChange),
      onKeyDown: callAllHandlers(rest.onKeyDown, onKeyDown),
      onFocus: callAllHandlers(rest.onFocus, onFocus),
      onBlur: callAllHandlers(rest.onBlur, onBlur),
      value: values[index] || "",
      autoComplete: otp ? "one-time-code" : "off",
      placeholder: hasFocus ? "" : placeholder
    });
  }, [descendants, focusedIndex, getNextValue, id, isDisabled, mask, isInvalid, manageFocus, onComplete, otp, placeholder, setValue, setValues, type, values]);
  return {
    // prop getter
    getInputProps: getInputProps,
    // state
    id: id,
    descendants: descendants,
    values: values,
    // actions
    setValue: setValue,
    setValues: setValues,
    clear: clear
  };
}

/**
 * @internal
 */
function usePinInputField(props, ref) {
  if (props === void 0) {
    props = {};
  }

  if (ref === void 0) {
    ref = null;
  }

  var _usePinInputContext = usePinInputContext(),
      getInputProps = _usePinInputContext.getInputProps;

  var _usePinInputDescendan = usePinInputDescendant(),
      index = _usePinInputDescendan.index,
      register = _usePinInputDescendan.register;

  return getInputProps(_extends({}, props, {
    ref: mergeRefs(register, ref),
    index: index
  }));
}

var _excluded = ["children"],
    _excluded2 = ["descendants"];
var PinInput = function PinInput(props) {
  var styles = useStyleConfig("PinInput", props);

  var _omitThemingProps = omitThemingProps(props),
      children = _omitThemingProps.children,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var _usePinInput = usePinInput(rest),
      descendants = _usePinInput.descendants,
      context = _objectWithoutPropertiesLoose(_usePinInput, _excluded2);

  var clones = getValidChildren(children).map(function (child) {
    return /*#__PURE__*/React.cloneElement(child, {
      __css: styles
    });
  });
  return /*#__PURE__*/React.createElement(PinInputDescendantsProvider, {
    value: descendants
  }, /*#__PURE__*/React.createElement(PinInputProvider, {
    value: context
  }, clones));
};

if (__DEV__) {
  PinInput.displayName = "PinInput";
}

var PinInputField = /*#__PURE__*/forwardRef(function (props, ref) {
  var inputProps = usePinInputField(props, ref);
  return /*#__PURE__*/React.createElement(chakra.input, _extends({}, inputProps, {
    className: cx("chakra-pin-input", props.className)
  }));
});

if (__DEV__) {
  PinInputField.displayName = "PinInputField";
}

export { PinInput, PinInputDescendantsProvider, PinInputField, PinInputProvider, usePinInput, usePinInputContext, usePinInputDescendant, usePinInputDescendants, usePinInputDescendantsContext, usePinInputField };
