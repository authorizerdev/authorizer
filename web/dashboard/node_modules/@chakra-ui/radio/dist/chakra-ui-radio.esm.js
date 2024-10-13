import { forwardRef, chakra, useMultiStyleConfig, omitThemingProps, layoutPropNames } from '@chakra-ui/system';
import { isInputEvent, cx, __DEV__, warn, dataAttr, callAllHandlers, ariaAttr, callAll, split } from '@chakra-ui/utils';
import * as React from 'react';
import { useState, useCallback } from 'react';
import { mergeRefs, createContext } from '@chakra-ui/react-utils';
import { useControllableProp, useId, useBoolean } from '@chakra-ui/hooks';
import { useFormControlContext } from '@chakra-ui/form-control';
import { visuallyHiddenStyle } from '@chakra-ui/visually-hidden';

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

var _excluded$3 = ["onChange", "value", "defaultValue", "name", "isDisabled", "isFocusable", "isNative"];

/**
 * React hook to manage a group of radio inputs
 */
function useRadioGroup(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      onChangeProp = _props.onChange,
      valueProp = _props.value,
      defaultValue = _props.defaultValue,
      nameProp = _props.name,
      isDisabled = _props.isDisabled,
      isFocusable = _props.isFocusable,
      isNative = _props.isNative,
      htmlProps = _objectWithoutPropertiesLoose(_props, _excluded$3);

  var _React$useState = React.useState(defaultValue || ""),
      valueState = _React$useState[0],
      setValue = _React$useState[1];

  var _useControllableProp = useControllableProp(valueProp, valueState),
      isControlled = _useControllableProp[0],
      value = _useControllableProp[1];

  var ref = React.useRef(null);
  var focus = React.useCallback(function () {
    var rootNode = ref.current;
    if (!rootNode) return;
    var query = "input:not(:disabled):checked";
    var firstEnabledAndCheckedInput = rootNode.querySelector(query);

    if (firstEnabledAndCheckedInput) {
      firstEnabledAndCheckedInput.focus();
      return;
    }

    query = "input:not(:disabled)";
    var firstEnabledInput = rootNode.querySelector(query);
    firstEnabledInput == null ? void 0 : firstEnabledInput.focus();
  }, []);
  /**
   * All radio options must use the same name
   */

  var fallbackName = useId(undefined, "radio");
  var name = nameProp || fallbackName;
  var onChange = React.useCallback(function (eventOrValue) {
    var nextValue = isInputEvent(eventOrValue) ? eventOrValue.target.value : eventOrValue;

    if (!isControlled) {
      setValue(nextValue);
    }

    onChangeProp == null ? void 0 : onChangeProp(String(nextValue));
  }, [onChangeProp, isControlled]);
  var getRootProps = React.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: mergeRefs(forwardedRef, ref),
      role: "radiogroup"
    });
  }, []);
  var getRadioProps = React.useCallback(function (props, ref) {
    var _extends2;

    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var checkedKey = isNative ? "checked" : "isChecked";
    return _extends({}, props, (_extends2 = {
      ref: ref,
      name: name
    }, _extends2[checkedKey] = value != null ? props.value === value : undefined, _extends2.onChange = onChange, _extends2["data-radiogroup"] = true, _extends2));
  }, [isNative, name, onChange, value]);
  return {
    getRootProps: getRootProps,
    getRadioProps: getRadioProps,
    name: name,
    ref: ref,
    focus: focus,
    setValue: setValue,
    value: value,
    onChange: onChange,
    isDisabled: isDisabled,
    isFocusable: isFocusable,
    htmlProps: htmlProps
  };
}

var _excluded$2 = ["colorScheme", "size", "variant", "children", "className", "isDisabled", "isFocusable"];

var _createContext = createContext({
  name: "RadioGroupContext",
  strict: false
}),
    RadioGroupProvider = _createContext[0],
    useRadioGroupContext = _createContext[1];

/**
 * Used for multiple radios which are bound in one group,
 * and it indicates which option is selected.
 *
 * @see Docs https://chakra-ui.com/radio
 */
var RadioGroup = /*#__PURE__*/forwardRef(function (props, ref) {
  var colorScheme = props.colorScheme,
      size = props.size,
      variant = props.variant,
      children = props.children,
      className = props.className,
      isDisabled = props.isDisabled,
      isFocusable = props.isFocusable,
      rest = _objectWithoutPropertiesLoose(props, _excluded$2);

  var _useRadioGroup = useRadioGroup(rest),
      value = _useRadioGroup.value,
      onChange = _useRadioGroup.onChange,
      getRootProps = _useRadioGroup.getRootProps,
      name = _useRadioGroup.name,
      htmlProps = _useRadioGroup.htmlProps;

  var group = React.useMemo(function () {
    return {
      name: name,
      size: size,
      onChange: onChange,
      colorScheme: colorScheme,
      value: value,
      variant: variant,
      isDisabled: isDisabled,
      isFocusable: isFocusable
    };
  }, [name, size, onChange, colorScheme, value, variant, isDisabled, isFocusable]);
  var groupProps = getRootProps(htmlProps, ref);

  var _className = cx("chakra-radio-group", className);

  return /*#__PURE__*/React.createElement(RadioGroupProvider, {
    value: group
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({}, groupProps, {
    className: _className
  }), children));
});

if (__DEV__) {
  RadioGroup.displayName = "RadioGroup";
}

var _excluded$1 = ["defaultIsChecked", "defaultChecked", "isChecked", "isFocusable", "isDisabled", "isReadOnly", "isRequired", "onChange", "isInvalid", "name", "value", "id", "data-radiogroup"];
/**
 * @todo use the `useClickable` hook here
 * to manage the isFocusable & isDisabled props
 */

function useRadio(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      defaultIsChecked = _props.defaultIsChecked,
      _props$defaultChecked = _props.defaultChecked,
      defaultChecked = _props$defaultChecked === void 0 ? defaultIsChecked : _props$defaultChecked,
      isCheckedProp = _props.isChecked,
      isFocusable = _props.isFocusable,
      isDisabledProp = _props.isDisabled,
      isReadOnlyProp = _props.isReadOnly,
      isRequiredProp = _props.isRequired,
      onChange = _props.onChange,
      isInvalidProp = _props.isInvalid,
      name = _props.name,
      value = _props.value,
      idProp = _props.id,
      dataRadioGroup = _props["data-radiogroup"],
      htmlProps = _objectWithoutPropertiesLoose(_props, _excluded$1);

  var uuid = useId(undefined, "radio");
  var formControl = useFormControlContext();
  var group = useRadioGroupContext();
  var isWithinRadioGroup = !!group || !!dataRadioGroup;
  var isWithinFormControl = !!formControl;
  var id = isWithinFormControl && !isWithinRadioGroup ? formControl.id : uuid;
  id = idProp != null ? idProp : id;
  var isDisabled = isDisabledProp != null ? isDisabledProp : formControl == null ? void 0 : formControl.isDisabled;
  var isReadOnly = isReadOnlyProp != null ? isReadOnlyProp : formControl == null ? void 0 : formControl.isReadOnly;
  var isRequired = isRequiredProp != null ? isRequiredProp : formControl == null ? void 0 : formControl.isRequired;
  var isInvalid = isInvalidProp != null ? isInvalidProp : formControl == null ? void 0 : formControl.isInvalid;

  var _useBoolean = useBoolean(),
      isFocused = _useBoolean[0],
      setFocused = _useBoolean[1];

  var _useBoolean2 = useBoolean(),
      isHovered = _useBoolean2[0],
      setHovering = _useBoolean2[1];

  var _useBoolean3 = useBoolean(),
      isActive = _useBoolean3[0],
      setActive = _useBoolean3[1];

  var _useState = useState(Boolean(defaultChecked)),
      isCheckedState = _useState[0],
      setChecked = _useState[1];

  var _useControllableProp = useControllableProp(isCheckedProp, isCheckedState),
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
      setChecked(event.target.checked);
    }

    onChange == null ? void 0 : onChange(event);
  }, [isControlled, isDisabled, isReadOnly, onChange]);
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
  var getCheckboxProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      "data-active": dataAttr(isActive),
      "data-hover": dataAttr(isHovered),
      "data-disabled": dataAttr(isDisabled),
      "data-invalid": dataAttr(isInvalid),
      "data-checked": dataAttr(isChecked),
      "data-focus": dataAttr(isFocused),
      "data-readonly": dataAttr(isReadOnly),
      "aria-hidden": true,
      onMouseDown: callAllHandlers(props.onMouseDown, setActive.on),
      onMouseUp: callAllHandlers(props.onMouseUp, setActive.off),
      onMouseEnter: callAllHandlers(props.onMouseEnter, setHovering.on),
      onMouseLeave: callAllHandlers(props.onMouseLeave, setHovering.off)
    });
  }, [isActive, isHovered, isDisabled, isInvalid, isChecked, isFocused, isReadOnly, setActive.on, setActive.off, setHovering.on, setHovering.off]);

  var _ref = formControl != null ? formControl : {},
      onFocus = _ref.onFocus,
      onBlur = _ref.onBlur;

  var getInputProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var trulyDisabled = isDisabled && !isFocusable;
    return _extends({}, props, {
      id: id,
      ref: ref,
      type: "radio",
      name: name,
      value: value,
      onChange: callAllHandlers(props.onChange, handleChange),
      onBlur: callAllHandlers(onBlur, props.onBlur, setFocused.off),
      onFocus: callAllHandlers(onFocus, props.onFocus, setFocused.on),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown),
      onKeyUp: callAllHandlers(props.onKeyUp, onKeyUp),
      checked: isChecked,
      disabled: trulyDisabled,
      readOnly: isReadOnly,
      required: isRequired,
      "aria-invalid": ariaAttr(isInvalid),
      "aria-disabled": ariaAttr(trulyDisabled),
      "aria-required": ariaAttr(isRequired),
      "data-readonly": dataAttr(isReadOnly),
      style: visuallyHiddenStyle
    });
  }, [isDisabled, isFocusable, id, name, value, handleChange, onBlur, setFocused, onFocus, onKeyDown, onKeyUp, isChecked, isReadOnly, isRequired, isInvalid]);

  var getLabelProps = function getLabelProps(props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      onMouseDown: callAllHandlers(props.onMouseDown, stop),
      onTouchStart: callAllHandlers(props.onTouchStart, stop),
      "data-disabled": dataAttr(isDisabled),
      "data-checked": dataAttr(isChecked),
      "data-invalid": dataAttr(isInvalid)
    });
  };

  var getRootProps = function getRootProps(props, ref) {
    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      "data-disabled": dataAttr(isDisabled),
      "data-checked": dataAttr(isChecked),
      "data-invalid": dataAttr(isInvalid)
    });
  };

  return {
    state: {
      isInvalid: isInvalid,
      isFocused: isFocused,
      isChecked: isChecked,
      isActive: isActive,
      isHovered: isHovered,
      isDisabled: isDisabled,
      isReadOnly: isReadOnly,
      isRequired: isRequired
    },
    getCheckboxProps: getCheckboxProps,
    getInputProps: getInputProps,
    getLabelProps: getLabelProps,
    getRootProps: getRootProps,
    htmlProps: htmlProps
  };
}
/**
 * Prevent `onBlur` being fired when the checkbox label is touched
 */

function stop(event) {
  event.preventDefault();
  event.stopPropagation();
}

var _excluded = ["spacing", "children", "isFullWidth", "isDisabled", "isFocusable"];

/**
 * Radio component is used in forms when a user needs to select a single value from
 * several options.
 *
 * @see Docs https://chakra-ui.com/radio
 */
var Radio = /*#__PURE__*/forwardRef(function (props, ref) {
  var _props$name;

  var group = useRadioGroupContext();
  var onChangeProp = props.onChange,
      valueProp = props.value;
  var styles = useMultiStyleConfig("Radio", _extends({}, group, props));
  var ownProps = omitThemingProps(props);

  var _ownProps$spacing = ownProps.spacing,
      spacing = _ownProps$spacing === void 0 ? "0.5rem" : _ownProps$spacing,
      children = ownProps.children,
      isFullWidth = ownProps.isFullWidth,
      _ownProps$isDisabled = ownProps.isDisabled,
      isDisabled = _ownProps$isDisabled === void 0 ? group == null ? void 0 : group.isDisabled : _ownProps$isDisabled,
      _ownProps$isFocusable = ownProps.isFocusable,
      isFocusable = _ownProps$isFocusable === void 0 ? group == null ? void 0 : group.isFocusable : _ownProps$isFocusable,
      rest = _objectWithoutPropertiesLoose(ownProps, _excluded);

  var isChecked = props.isChecked;

  if ((group == null ? void 0 : group.value) != null && valueProp != null) {
    isChecked = group.value === valueProp;
  }

  var onChange = onChangeProp;

  if (group != null && group.onChange && valueProp != null) {
    onChange = callAll(group.onChange, onChangeProp);
  }

  var name = (_props$name = props == null ? void 0 : props.name) != null ? _props$name : group == null ? void 0 : group.name;

  var _useRadio = useRadio(_extends({}, rest, {
    isChecked: isChecked,
    isFocusable: isFocusable,
    isDisabled: isDisabled,
    onChange: onChange,
    name: name
  })),
      getInputProps = _useRadio.getInputProps,
      getCheckboxProps = _useRadio.getCheckboxProps,
      getLabelProps = _useRadio.getLabelProps,
      getRootProps = _useRadio.getRootProps,
      htmlProps = _useRadio.htmlProps;

  var _split = split(htmlProps, layoutPropNames),
      layoutProps = _split[0],
      otherProps = _split[1];

  var checkboxProps = getCheckboxProps(otherProps);
  var inputProps = getInputProps({}, ref);
  var labelProps = getLabelProps();
  var rootProps = Object.assign({}, layoutProps, getRootProps());

  var rootStyles = _extends({
    width: isFullWidth ? "full" : undefined,
    display: "inline-flex",
    alignItems: "center",
    verticalAlign: "top",
    cursor: "pointer"
  }, styles.container);

  var checkboxStyles = _extends({
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0
  }, styles.control);

  var labelStyles = _extends({
    userSelect: "none",
    marginStart: spacing
  }, styles.label);

  return /*#__PURE__*/React.createElement(chakra.label, _extends({
    className: "chakra-radio"
  }, rootProps, {
    __css: rootStyles
  }), /*#__PURE__*/React.createElement("input", _extends({
    className: "chakra-radio__input"
  }, inputProps)), /*#__PURE__*/React.createElement(chakra.span, _extends({
    className: "chakra-radio__control"
  }, checkboxProps, {
    __css: checkboxStyles
  })), children && /*#__PURE__*/React.createElement(chakra.span, _extends({
    className: "chakra-radio__label"
  }, labelProps, {
    __css: labelStyles
  }), children));
});

if (__DEV__) {
  Radio.displayName = "Radio";
}

export { Radio, RadioGroup, useRadio, useRadioGroup, useRadioGroupContext };
