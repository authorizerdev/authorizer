import { isInputEvent, addItem, removeItem, __DEV__, warn, dataAttr, callAllHandlers, focus, callAll, cx } from '@chakra-ui/utils';
import { createContext, mergeRefs } from '@chakra-ui/react-utils';
import * as React from 'react';
import { useCallback, useRef, useState } from 'react';
import { useCallbackRef, useControllableState, useBoolean, useControllableProp, useSafeLayoutEffect } from '@chakra-ui/hooks';
import { chakra, forwardRef, useMultiStyleConfig, omitThemingProps } from '@chakra-ui/system';
import { motion, AnimatePresence } from 'framer-motion';
import { visuallyHiddenStyle } from '@chakra-ui/visually-hidden';

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

/**
 * React hook that provides all the state management logic
 * for a group of checkboxes.
 *
 * It is consumed by the `CheckboxGroup` component
 */
function useCheckboxGroup(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      defaultValue = _props.defaultValue,
      valueProp = _props.value,
      onChange = _props.onChange,
      isDisabled = _props.isDisabled,
      isNative = _props.isNative;
  var onChangeProp = useCallbackRef(onChange);

  var _useControllableState = useControllableState({
    value: valueProp,
    defaultValue: defaultValue || [],
    onChange: onChangeProp
  }),
      value = _useControllableState[0],
      setValue = _useControllableState[1];

  var handleChange = useCallback(function (eventOrValue) {
    if (!value) return;
    var isChecked = isInputEvent(eventOrValue) ? eventOrValue.target.checked : !value.includes(eventOrValue);
    var selectedValue = isInputEvent(eventOrValue) ? eventOrValue.target.value : eventOrValue;
    var nextValue = isChecked ? addItem(value, selectedValue) : removeItem(value, selectedValue);
    setValue(nextValue);
  }, [setValue, value]);
  var getCheckboxProps = useCallback(function (props) {
    var _extends2;

    if (props === void 0) {
      props = {};
    }

    var checkedKey = isNative ? "checked" : "isChecked";
    return _extends({}, props, (_extends2 = {}, _extends2[checkedKey] = value.includes(props.value), _extends2.onChange = handleChange, _extends2));
  }, [handleChange, isNative, value]);
  return {
    value: value,
    isDisabled: isDisabled,
    onChange: handleChange,
    setValue: setValue,
    getCheckboxProps: getCheckboxProps
  };
}

var _createContext = createContext({
  name: "CheckboxGroupContext",
  strict: false
}),
    CheckboxGroupProvider = _createContext[0],
    useCheckboxGroupContext = _createContext[1];
/**
 * Used for multiple checkboxes which are bound in one group,
 * and it indicates whether one or more options are selected.
 *
 * @see Docs https://chakra-ui.com/checkbox
 */

var CheckboxGroup = function CheckboxGroup(props) {
  var colorScheme = props.colorScheme,
      size = props.size,
      variant = props.variant,
      children = props.children,
      isDisabled = props.isDisabled;

  var _useCheckboxGroup = useCheckboxGroup(props),
      value = _useCheckboxGroup.value,
      onChange = _useCheckboxGroup.onChange;

  var group = React.useMemo(function () {
    return {
      size: size,
      onChange: onChange,
      colorScheme: colorScheme,
      value: value,
      variant: variant,
      isDisabled: isDisabled
    };
  }, [size, onChange, colorScheme, value, variant, isDisabled]);
  return /*#__PURE__*/React.createElement(CheckboxGroupProvider, {
    value: group
  }, children);
};

if (__DEV__) {
  CheckboxGroup.displayName = "CheckboxGroup";
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

var _excluded$2 = ["isIndeterminate", "isChecked"];

var MotionSvg = "custom" in motion ? motion.custom(chakra.svg) : motion(chakra.svg);

var CheckIcon = function CheckIcon(props) {
  return /*#__PURE__*/React.createElement(MotionSvg, _extends({
    width: "1.2em",
    viewBox: "0 0 12 10",
    variants: {
      unchecked: {
        opacity: 0,
        strokeDashoffset: 16
      },
      checked: {
        opacity: 1,
        strokeDashoffset: 0,
        transition: {
          duration: 0.2
        }
      }
    },
    style: {
      fill: "none",
      strokeWidth: 2,
      stroke: "currentColor",
      strokeDasharray: 16
    }
  }, props), /*#__PURE__*/React.createElement("polyline", {
    points: "1.5 6 4.5 9 10.5 1"
  }));
};

var IndeterminateIcon = function IndeterminateIcon(props) {
  return /*#__PURE__*/React.createElement(MotionSvg, _extends({
    width: "1.2em",
    viewBox: "0 0 24 24",
    variants: {
      unchecked: {
        scaleX: 0.65,
        opacity: 0
      },
      checked: {
        scaleX: 1,
        opacity: 1,
        transition: {
          scaleX: {
            duration: 0
          },
          opacity: {
            duration: 0.02
          }
        }
      }
    },
    style: {
      stroke: "currentColor",
      strokeWidth: 4
    }
  }, props), /*#__PURE__*/React.createElement("line", {
    x1: "21",
    x2: "3",
    y1: "12",
    y2: "12"
  }));
};

var CheckboxTransition = function CheckboxTransition(_ref) {
  var open = _ref.open,
      children = _ref.children;
  return /*#__PURE__*/React.createElement(AnimatePresence, {
    initial: false
  }, open && /*#__PURE__*/React.createElement(motion.div, {
    variants: {
      unchecked: {
        scale: 0.5
      },
      checked: {
        scale: 1
      }
    },
    initial: "unchecked",
    animate: "checked",
    exit: "unchecked",
    style: {
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      height: "100%"
    }
  }, children));
};

/**
 * CheckboxIcon is used to visually indicate the checked or indeterminate
 * state of a checkbox.
 *
 * @todo allow users pass their own icon svgs
 */
var CheckboxIcon = function CheckboxIcon(props) {
  var isIndeterminate = props.isIndeterminate,
      isChecked = props.isChecked,
      rest = _objectWithoutPropertiesLoose(props, _excluded$2);

  var IconEl = isIndeterminate ? IndeterminateIcon : CheckIcon;
  return /*#__PURE__*/React.createElement(CheckboxTransition, {
    open: isChecked || isIndeterminate
  }, /*#__PURE__*/React.createElement(IconEl, rest));
};

var _excluded$1 = ["defaultIsChecked", "defaultChecked", "isChecked", "isFocusable", "isDisabled", "isReadOnly", "isRequired", "onChange", "isIndeterminate", "isInvalid", "name", "value", "id", "onBlur", "onFocus", "tabIndex", "aria-label", "aria-labelledby", "aria-invalid", "aria-describedby"];

/**
 * useCheckbox that provides all the state and focus management logic
 * for a checkbox. It is consumed by the `Checkbox` component
 *
 * @see Docs https://chakra-ui.com/checkbox#hooks
 */
function useCheckbox(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      defaultIsChecked = _props.defaultIsChecked,
      _props$defaultChecked = _props.defaultChecked,
      defaultChecked = _props$defaultChecked === void 0 ? defaultIsChecked : _props$defaultChecked,
      checkedProp = _props.isChecked,
      isFocusable = _props.isFocusable,
      isDisabled = _props.isDisabled,
      isReadOnly = _props.isReadOnly,
      isRequired = _props.isRequired,
      onChange = _props.onChange,
      isIndeterminate = _props.isIndeterminate,
      isInvalid = _props.isInvalid,
      name = _props.name,
      value = _props.value,
      id = _props.id,
      onBlur = _props.onBlur,
      onFocus = _props.onFocus,
      _props$tabIndex = _props.tabIndex,
      tabIndex = _props$tabIndex === void 0 ? undefined : _props$tabIndex,
      ariaLabel = _props["aria-label"],
      ariaLabelledBy = _props["aria-labelledby"],
      ariaInvalid = _props["aria-invalid"],
      ariaDescribedBy = _props["aria-describedby"],
      htmlProps = _objectWithoutPropertiesLoose(_props, _excluded$1);

  var onChangeProp = useCallbackRef(onChange);
  var onBlurProp = useCallbackRef(onBlur);
  var onFocusProp = useCallbackRef(onFocus);

  var _useBoolean = useBoolean(),
      isFocused = _useBoolean[0],
      setFocused = _useBoolean[1];

  var _useBoolean2 = useBoolean(),
      isHovered = _useBoolean2[0],
      setHovered = _useBoolean2[1];

  var _useBoolean3 = useBoolean(),
      isActive = _useBoolean3[0],
      setActive = _useBoolean3[1];

  var inputRef = useRef(null);

  var _useState = useState(true),
      rootIsLabelElement = _useState[0],
      setRootIsLabelElement = _useState[1];

  var _useState2 = useState(!!defaultChecked),
      checkedState = _useState2[0],
      setCheckedState = _useState2[1];

  var _useControllableProp = useControllableProp(checkedProp, checkedState),
      isControlled = _useControllableProp[0],
      isChecked = _useControllableProp[1];

  warn({
    condition: !!defaultIsChecked,
    message: 'The "defaultIsChecked" prop has been deprecated and will be removed in a future version. ' + 'Please use the "defaultChecked" prop instead, which mirrors default React checkbox behavior.'
  });
  var handleChange = useCallback(function (event) {
    if (isReadOnly || isDisabled) {
      event.preventDefault();
      return;
    }

    if (!isControlled) {
      if (isChecked) {
        setCheckedState(event.target.checked);
      } else {
        setCheckedState(isIndeterminate ? true : event.target.checked);
      }
    }

    onChangeProp == null ? void 0 : onChangeProp(event);
  }, [isReadOnly, isDisabled, isChecked, isControlled, isIndeterminate, onChangeProp]);
  useSafeLayoutEffect(function () {
    if (inputRef.current) {
      inputRef.current.indeterminate = Boolean(isIndeterminate);
    }
  }, [isIndeterminate]);
  var trulyDisabled = isDisabled && !isFocusable;
  var onKeyDown = useCallback(function (event) {
    if (event.key === " ") {
      setActive.on();
    }
  }, [setActive]);
  var onKeyUp = useCallback(function (event) {
    if (event.key === " ") {
      setActive.off();
    }
  }, [setActive]);
  /**
   * Sync state with uncontrolled form libraries like `react-hook-form`.
   *
   * These libraries set the checked value for input fields
   * using their refs. For the checkbox, it sets `ref.current.checked = true | false` directly.
   *
   * This means the `isChecked` state will get out of sync with `ref.current.checked`,
   * even though the input validation with work, the UI will not be up to date.
   *
   * Let's correct that by checking and syncing the state accordingly.
   */

  useSafeLayoutEffect(function () {
    if (!inputRef.current) return;
    var notInSync = inputRef.current.checked !== isChecked;

    if (notInSync) {
      setCheckedState(inputRef.current.checked);
    }
  }, [inputRef.current]);
  var getCheckboxProps = useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    var onPressDown = function onPressDown(event) {
      // On mousedown, the input blurs and returns focus to the `body`,
      // we need to prevent this. Native checkboxes keeps focus on `input`
      event.preventDefault();
      setActive.on();
    };

    return _extends({}, props, {
      ref: forwardedRef,
      "data-active": dataAttr(isActive),
      "data-hover": dataAttr(isHovered),
      "data-checked": dataAttr(isChecked),
      "data-focus": dataAttr(isFocused),
      "data-indeterminate": dataAttr(isIndeterminate),
      "data-disabled": dataAttr(isDisabled),
      "data-invalid": dataAttr(isInvalid),
      "data-readonly": dataAttr(isReadOnly),
      "aria-hidden": true,
      onMouseDown: callAllHandlers(props.onMouseDown, onPressDown),
      onMouseUp: callAllHandlers(props.onMouseUp, setActive.off),
      onMouseEnter: callAllHandlers(props.onMouseEnter, setHovered.on),
      onMouseLeave: callAllHandlers(props.onMouseLeave, setHovered.off)
    });
  }, [isActive, isChecked, isDisabled, isFocused, isHovered, isIndeterminate, isInvalid, isReadOnly, setActive, setHovered.off, setHovered.on]);
  var getRootProps = useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, htmlProps, props, {
      ref: mergeRefs(forwardedRef, function (node) {
        if (!node) return;
        setRootIsLabelElement(node.tagName === "LABEL");
      }),
      onClick: callAllHandlers(props.onClick, function () {
        /**
         * Accessibility:
         *
         * Ideally, `getRootProps` should be spread unto a `label` element.
         *
         * If the element was changed using the `as` prop or changing
         * the dom node `getRootProps` is spread unto (to a `div` or `span`), we'll trigger
         * click on the input when the element is clicked.
         * @see Issue https://github.com/chakra-ui/chakra-ui/issues/3480
         */
        if (!rootIsLabelElement) {
          var _inputRef$current;

          (_inputRef$current = inputRef.current) == null ? void 0 : _inputRef$current.click();
          focus(inputRef.current, {
            nextTick: true
          });
        }
      }),
      "data-disabled": dataAttr(isDisabled),
      "data-checked": dataAttr(isChecked),
      "data-invalid": dataAttr(isInvalid)
    });
  }, [htmlProps, isDisabled, isChecked, isInvalid, rootIsLabelElement]);
  var getInputProps = useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: mergeRefs(inputRef, forwardedRef),
      type: "checkbox",
      name: name,
      value: value,
      id: id,
      tabIndex: tabIndex,
      onChange: callAllHandlers(props.onChange, handleChange),
      onBlur: callAllHandlers(props.onBlur, onBlurProp, setFocused.off),
      onFocus: callAllHandlers(props.onFocus, onFocusProp, setFocused.on),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown),
      onKeyUp: callAllHandlers(props.onKeyUp, onKeyUp),
      required: isRequired,
      checked: isChecked,
      disabled: trulyDisabled,
      readOnly: isReadOnly,
      "aria-label": ariaLabel,
      "aria-labelledby": ariaLabelledBy,
      "aria-invalid": ariaInvalid ? Boolean(ariaInvalid) : isInvalid,
      "aria-describedby": ariaDescribedBy,
      "aria-disabled": isDisabled,
      style: visuallyHiddenStyle
    });
  }, [name, value, id, handleChange, setFocused.off, setFocused.on, onBlurProp, onFocusProp, onKeyDown, onKeyUp, isRequired, isChecked, trulyDisabled, isReadOnly, ariaLabel, ariaLabelledBy, ariaInvalid, isInvalid, ariaDescribedBy, isDisabled, tabIndex]);
  var getLabelProps = useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: forwardedRef,
      onMouseDown: callAllHandlers(props.onMouseDown, stopEvent),
      onTouchStart: callAllHandlers(props.onTouchStart, stopEvent),
      "data-disabled": dataAttr(isDisabled),
      "data-checked": dataAttr(isChecked),
      "data-invalid": dataAttr(isInvalid)
    });
  }, [isChecked, isDisabled, isInvalid]);
  return {
    state: {
      isInvalid: isInvalid,
      isFocused: isFocused,
      isChecked: isChecked,
      isActive: isActive,
      isHovered: isHovered,
      isIndeterminate: isIndeterminate,
      isDisabled: isDisabled,
      isReadOnly: isReadOnly,
      isRequired: isRequired
    },
    getRootProps: getRootProps,
    getCheckboxProps: getCheckboxProps,
    getInputProps: getInputProps,
    getLabelProps: getLabelProps,
    htmlProps: htmlProps
  };
}
/**
 * Prevent `onBlur` being fired when the checkbox label is touched
 */

function stopEvent(event) {
  event.preventDefault();
  event.stopPropagation();
}

var _excluded = ["spacing", "className", "children", "iconColor", "iconSize", "icon", "isChecked", "isDisabled", "onChange"];
var CheckboxControl = chakra("span", {
  baseStyle: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    verticalAlign: "top",
    userSelect: "none",
    flexShrink: 0
  }
});
var Label = chakra("label", {
  baseStyle: {
    cursor: "pointer",
    display: "inline-flex",
    alignItems: "center",
    verticalAlign: "top",
    position: "relative",
    _disabled: {
      cursor: "not-allowed"
    }
  }
});

/**
 * Checkbox
 *
 * React component used in forms when a user needs to select
 * multiple values from several options.
 *
 * @see Docs https://chakra-ui.com/checkbox
 */
var Checkbox = /*#__PURE__*/forwardRef(function (props, ref) {
  var group = useCheckboxGroupContext();

  var mergedProps = _extends({}, group, props);

  var styles = useMultiStyleConfig("Checkbox", mergedProps);
  var ownProps = omitThemingProps(props);

  var _ownProps$spacing = ownProps.spacing,
      spacing = _ownProps$spacing === void 0 ? "0.5rem" : _ownProps$spacing,
      className = ownProps.className,
      children = ownProps.children,
      iconColor = ownProps.iconColor,
      iconSize = ownProps.iconSize,
      _ownProps$icon = ownProps.icon,
      icon = _ownProps$icon === void 0 ? /*#__PURE__*/React.createElement(CheckboxIcon, null) : _ownProps$icon,
      isCheckedProp = ownProps.isChecked,
      _ownProps$isDisabled = ownProps.isDisabled,
      isDisabled = _ownProps$isDisabled === void 0 ? group == null ? void 0 : group.isDisabled : _ownProps$isDisabled,
      onChangeProp = ownProps.onChange,
      rest = _objectWithoutPropertiesLoose(ownProps, _excluded);

  var isChecked = isCheckedProp;

  if (group != null && group.value && ownProps.value) {
    isChecked = group.value.includes(ownProps.value);
  }

  var onChange = onChangeProp;

  if (group != null && group.onChange && ownProps.value) {
    onChange = callAll(group.onChange, onChangeProp);
  }

  var _useCheckbox = useCheckbox(_extends({}, rest, {
    isDisabled: isDisabled,
    isChecked: isChecked,
    onChange: onChange
  })),
      state = _useCheckbox.state,
      getInputProps = _useCheckbox.getInputProps,
      getCheckboxProps = _useCheckbox.getCheckboxProps,
      getLabelProps = _useCheckbox.getLabelProps,
      getRootProps = _useCheckbox.getRootProps;

  var iconStyles = React.useMemo(function () {
    return _extends({
      opacity: state.isChecked || state.isIndeterminate ? 1 : 0,
      transform: state.isChecked || state.isIndeterminate ? "scale(1)" : "scale(0.95)",
      fontSize: iconSize,
      color: iconColor
    }, styles.icon);
  }, [iconColor, iconSize, state.isChecked, state.isIndeterminate, styles.icon]);
  var clonedIcon = /*#__PURE__*/React.cloneElement(icon, {
    __css: iconStyles,
    isIndeterminate: state.isIndeterminate,
    isChecked: state.isChecked
  });
  return /*#__PURE__*/React.createElement(Label, _extends({
    __css: styles.container,
    className: cx("chakra-checkbox", className)
  }, getRootProps()), /*#__PURE__*/React.createElement("input", _extends({
    className: "chakra-checkbox__input"
  }, getInputProps({}, ref))), /*#__PURE__*/React.createElement(CheckboxControl, _extends({
    __css: styles.control,
    className: "chakra-checkbox__control"
  }, getCheckboxProps()), clonedIcon), children && /*#__PURE__*/React.createElement(chakra.span, _extends({
    className: "chakra-checkbox__label"
  }, getLabelProps(), {
    __css: _extends({
      marginStart: spacing
    }, styles.label)
  }), children));
});

if (__DEV__) {
  Checkbox.displayName = "Checkbox";
}

export { Checkbox, CheckboxGroup, useCheckbox, useCheckboxGroup, useCheckboxGroupContext };
