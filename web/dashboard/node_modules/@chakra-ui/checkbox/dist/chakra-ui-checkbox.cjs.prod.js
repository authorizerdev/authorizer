'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
var React = require('react');
var hooks = require('@chakra-ui/hooks');
var system = require('@chakra-ui/system');
var framerMotion = require('framer-motion');
var visuallyHidden = require('@chakra-ui/visually-hidden');

function _interopNamespace(e) {
  if (e && e.__esModule) return e;
  var n = Object.create(null);
  if (e) {
    Object.keys(e).forEach(function (k) {
      if (k !== 'default') {
        var d = Object.getOwnPropertyDescriptor(e, k);
        Object.defineProperty(n, k, d.get ? d : {
          enumerable: true,
          get: function () { return e[k]; }
        });
      }
    });
  }
  n["default"] = e;
  return Object.freeze(n);
}

var React__namespace = /*#__PURE__*/_interopNamespace(React);

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
  var onChangeProp = hooks.useCallbackRef(onChange);

  var _useControllableState = hooks.useControllableState({
    value: valueProp,
    defaultValue: defaultValue || [],
    onChange: onChangeProp
  }),
      value = _useControllableState[0],
      setValue = _useControllableState[1];

  var handleChange = React.useCallback(function (eventOrValue) {
    if (!value) return;
    var isChecked = utils.isInputEvent(eventOrValue) ? eventOrValue.target.checked : !value.includes(eventOrValue);
    var selectedValue = utils.isInputEvent(eventOrValue) ? eventOrValue.target.value : eventOrValue;
    var nextValue = isChecked ? utils.addItem(value, selectedValue) : utils.removeItem(value, selectedValue);
    setValue(nextValue);
  }, [setValue, value]);
  var getCheckboxProps = React.useCallback(function (props) {
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

var _createContext = reactUtils.createContext({
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

  var group = React__namespace.useMemo(function () {
    return {
      size: size,
      onChange: onChange,
      colorScheme: colorScheme,
      value: value,
      variant: variant,
      isDisabled: isDisabled
    };
  }, [size, onChange, colorScheme, value, variant, isDisabled]);
  return /*#__PURE__*/React__namespace.createElement(CheckboxGroupProvider, {
    value: group
  }, children);
};

if (utils.__DEV__) {
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

var MotionSvg = "custom" in framerMotion.motion ? framerMotion.motion.custom(system.chakra.svg) : framerMotion.motion(system.chakra.svg);

var CheckIcon = function CheckIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(MotionSvg, _extends({
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
  }, props), /*#__PURE__*/React__namespace.createElement("polyline", {
    points: "1.5 6 4.5 9 10.5 1"
  }));
};

var IndeterminateIcon = function IndeterminateIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(MotionSvg, _extends({
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
  }, props), /*#__PURE__*/React__namespace.createElement("line", {
    x1: "21",
    x2: "3",
    y1: "12",
    y2: "12"
  }));
};

var CheckboxTransition = function CheckboxTransition(_ref) {
  var open = _ref.open,
      children = _ref.children;
  return /*#__PURE__*/React__namespace.createElement(framerMotion.AnimatePresence, {
    initial: false
  }, open && /*#__PURE__*/React__namespace.createElement(framerMotion.motion.div, {
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
  return /*#__PURE__*/React__namespace.createElement(CheckboxTransition, {
    open: isChecked || isIndeterminate
  }, /*#__PURE__*/React__namespace.createElement(IconEl, rest));
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

  var onChangeProp = hooks.useCallbackRef(onChange);
  var onBlurProp = hooks.useCallbackRef(onBlur);
  var onFocusProp = hooks.useCallbackRef(onFocus);

  var _useBoolean = hooks.useBoolean(),
      isFocused = _useBoolean[0],
      setFocused = _useBoolean[1];

  var _useBoolean2 = hooks.useBoolean(),
      isHovered = _useBoolean2[0],
      setHovered = _useBoolean2[1];

  var _useBoolean3 = hooks.useBoolean(),
      isActive = _useBoolean3[0],
      setActive = _useBoolean3[1];

  var inputRef = React.useRef(null);

  var _useState = React.useState(true),
      rootIsLabelElement = _useState[0],
      setRootIsLabelElement = _useState[1];

  var _useState2 = React.useState(!!defaultChecked),
      checkedState = _useState2[0],
      setCheckedState = _useState2[1];

  var _useControllableProp = hooks.useControllableProp(checkedProp, checkedState),
      isControlled = _useControllableProp[0],
      isChecked = _useControllableProp[1];

  utils.warn({
    condition: !!defaultIsChecked,
    message: 'The "defaultIsChecked" prop has been deprecated and will be removed in a future version. ' + 'Please use the "defaultChecked" prop instead, which mirrors default React checkbox behavior.'
  });
  var handleChange = React.useCallback(function (event) {
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
  hooks.useSafeLayoutEffect(function () {
    if (inputRef.current) {
      inputRef.current.indeterminate = Boolean(isIndeterminate);
    }
  }, [isIndeterminate]);
  var trulyDisabled = isDisabled && !isFocusable;
  var onKeyDown = React.useCallback(function (event) {
    if (event.key === " ") {
      setActive.on();
    }
  }, [setActive]);
  var onKeyUp = React.useCallback(function (event) {
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

  hooks.useSafeLayoutEffect(function () {
    if (!inputRef.current) return;
    var notInSync = inputRef.current.checked !== isChecked;

    if (notInSync) {
      setCheckedState(inputRef.current.checked);
    }
  }, [inputRef.current]);
  var getCheckboxProps = React.useCallback(function (props, forwardedRef) {
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
      "data-active": utils.dataAttr(isActive),
      "data-hover": utils.dataAttr(isHovered),
      "data-checked": utils.dataAttr(isChecked),
      "data-focus": utils.dataAttr(isFocused),
      "data-indeterminate": utils.dataAttr(isIndeterminate),
      "data-disabled": utils.dataAttr(isDisabled),
      "data-invalid": utils.dataAttr(isInvalid),
      "data-readonly": utils.dataAttr(isReadOnly),
      "aria-hidden": true,
      onMouseDown: utils.callAllHandlers(props.onMouseDown, onPressDown),
      onMouseUp: utils.callAllHandlers(props.onMouseUp, setActive.off),
      onMouseEnter: utils.callAllHandlers(props.onMouseEnter, setHovered.on),
      onMouseLeave: utils.callAllHandlers(props.onMouseLeave, setHovered.off)
    });
  }, [isActive, isChecked, isDisabled, isFocused, isHovered, isIndeterminate, isInvalid, isReadOnly, setActive, setHovered.off, setHovered.on]);
  var getRootProps = React.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, htmlProps, props, {
      ref: reactUtils.mergeRefs(forwardedRef, function (node) {
        if (!node) return;
        setRootIsLabelElement(node.tagName === "LABEL");
      }),
      onClick: utils.callAllHandlers(props.onClick, function () {
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
          utils.focus(inputRef.current, {
            nextTick: true
          });
        }
      }),
      "data-disabled": utils.dataAttr(isDisabled),
      "data-checked": utils.dataAttr(isChecked),
      "data-invalid": utils.dataAttr(isInvalid)
    });
  }, [htmlProps, isDisabled, isChecked, isInvalid, rootIsLabelElement]);
  var getInputProps = React.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: reactUtils.mergeRefs(inputRef, forwardedRef),
      type: "checkbox",
      name: name,
      value: value,
      id: id,
      tabIndex: tabIndex,
      onChange: utils.callAllHandlers(props.onChange, handleChange),
      onBlur: utils.callAllHandlers(props.onBlur, onBlurProp, setFocused.off),
      onFocus: utils.callAllHandlers(props.onFocus, onFocusProp, setFocused.on),
      onKeyDown: utils.callAllHandlers(props.onKeyDown, onKeyDown),
      onKeyUp: utils.callAllHandlers(props.onKeyUp, onKeyUp),
      required: isRequired,
      checked: isChecked,
      disabled: trulyDisabled,
      readOnly: isReadOnly,
      "aria-label": ariaLabel,
      "aria-labelledby": ariaLabelledBy,
      "aria-invalid": ariaInvalid ? Boolean(ariaInvalid) : isInvalid,
      "aria-describedby": ariaDescribedBy,
      "aria-disabled": isDisabled,
      style: visuallyHidden.visuallyHiddenStyle
    });
  }, [name, value, id, handleChange, setFocused.off, setFocused.on, onBlurProp, onFocusProp, onKeyDown, onKeyUp, isRequired, isChecked, trulyDisabled, isReadOnly, ariaLabel, ariaLabelledBy, ariaInvalid, isInvalid, ariaDescribedBy, isDisabled, tabIndex]);
  var getLabelProps = React.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: forwardedRef,
      onMouseDown: utils.callAllHandlers(props.onMouseDown, stopEvent),
      onTouchStart: utils.callAllHandlers(props.onTouchStart, stopEvent),
      "data-disabled": utils.dataAttr(isDisabled),
      "data-checked": utils.dataAttr(isChecked),
      "data-invalid": utils.dataAttr(isInvalid)
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
var CheckboxControl = system.chakra("span", {
  baseStyle: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    verticalAlign: "top",
    userSelect: "none",
    flexShrink: 0
  }
});
var Label = system.chakra("label", {
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
var Checkbox = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var group = useCheckboxGroupContext();

  var mergedProps = _extends({}, group, props);

  var styles = system.useMultiStyleConfig("Checkbox", mergedProps);
  var ownProps = system.omitThemingProps(props);

  var _ownProps$spacing = ownProps.spacing,
      spacing = _ownProps$spacing === void 0 ? "0.5rem" : _ownProps$spacing,
      className = ownProps.className,
      children = ownProps.children,
      iconColor = ownProps.iconColor,
      iconSize = ownProps.iconSize,
      _ownProps$icon = ownProps.icon,
      icon = _ownProps$icon === void 0 ? /*#__PURE__*/React__namespace.createElement(CheckboxIcon, null) : _ownProps$icon,
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
    onChange = utils.callAll(group.onChange, onChangeProp);
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

  var iconStyles = React__namespace.useMemo(function () {
    return _extends({
      opacity: state.isChecked || state.isIndeterminate ? 1 : 0,
      transform: state.isChecked || state.isIndeterminate ? "scale(1)" : "scale(0.95)",
      fontSize: iconSize,
      color: iconColor
    }, styles.icon);
  }, [iconColor, iconSize, state.isChecked, state.isIndeterminate, styles.icon]);
  var clonedIcon = /*#__PURE__*/React__namespace.cloneElement(icon, {
    __css: iconStyles,
    isIndeterminate: state.isIndeterminate,
    isChecked: state.isChecked
  });
  return /*#__PURE__*/React__namespace.createElement(Label, _extends({
    __css: styles.container,
    className: utils.cx("chakra-checkbox", className)
  }, getRootProps()), /*#__PURE__*/React__namespace.createElement("input", _extends({
    className: "chakra-checkbox__input"
  }, getInputProps({}, ref))), /*#__PURE__*/React__namespace.createElement(CheckboxControl, _extends({
    __css: styles.control,
    className: "chakra-checkbox__control"
  }, getCheckboxProps()), clonedIcon), children && /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    className: "chakra-checkbox__label"
  }, getLabelProps(), {
    __css: _extends({
      marginStart: spacing
    }, styles.label)
  }), children));
});

if (utils.__DEV__) {
  Checkbox.displayName = "Checkbox";
}

exports.Checkbox = Checkbox;
exports.CheckboxGroup = CheckboxGroup;
exports.useCheckbox = useCheckbox;
exports.useCheckboxGroup = useCheckboxGroup;
exports.useCheckboxGroupContext = useCheckboxGroupContext;
