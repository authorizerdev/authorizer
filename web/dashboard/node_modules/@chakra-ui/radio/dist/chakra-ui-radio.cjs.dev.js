'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var React = require('react');
var reactUtils = require('@chakra-ui/react-utils');
var hooks = require('@chakra-ui/hooks');
var formControl = require('@chakra-ui/form-control');
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

  var _React$useState = React__namespace.useState(defaultValue || ""),
      valueState = _React$useState[0],
      setValue = _React$useState[1];

  var _useControllableProp = hooks.useControllableProp(valueProp, valueState),
      isControlled = _useControllableProp[0],
      value = _useControllableProp[1];

  var ref = React__namespace.useRef(null);
  var focus = React__namespace.useCallback(function () {
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

  var fallbackName = hooks.useId(undefined, "radio");
  var name = nameProp || fallbackName;
  var onChange = React__namespace.useCallback(function (eventOrValue) {
    var nextValue = utils.isInputEvent(eventOrValue) ? eventOrValue.target.value : eventOrValue;

    if (!isControlled) {
      setValue(nextValue);
    }

    onChangeProp == null ? void 0 : onChangeProp(String(nextValue));
  }, [onChangeProp, isControlled]);
  var getRootProps = React__namespace.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: reactUtils.mergeRefs(forwardedRef, ref),
      role: "radiogroup"
    });
  }, []);
  var getRadioProps = React__namespace.useCallback(function (props, ref) {
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

var _createContext = reactUtils.createContext({
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
var RadioGroup = /*#__PURE__*/system.forwardRef(function (props, ref) {
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

  var group = React__namespace.useMemo(function () {
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

  var _className = utils.cx("chakra-radio-group", className);

  return /*#__PURE__*/React__namespace.createElement(RadioGroupProvider, {
    value: group
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, groupProps, {
    className: _className
  }), children));
});

if (utils.__DEV__) {
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

  var uuid = hooks.useId(undefined, "radio");
  var formControl$1 = formControl.useFormControlContext();
  var group = useRadioGroupContext();
  var isWithinRadioGroup = !!group || !!dataRadioGroup;
  var isWithinFormControl = !!formControl$1;
  var id = isWithinFormControl && !isWithinRadioGroup ? formControl$1.id : uuid;
  id = idProp != null ? idProp : id;
  var isDisabled = isDisabledProp != null ? isDisabledProp : formControl$1 == null ? void 0 : formControl$1.isDisabled;
  var isReadOnly = isReadOnlyProp != null ? isReadOnlyProp : formControl$1 == null ? void 0 : formControl$1.isReadOnly;
  var isRequired = isRequiredProp != null ? isRequiredProp : formControl$1 == null ? void 0 : formControl$1.isRequired;
  var isInvalid = isInvalidProp != null ? isInvalidProp : formControl$1 == null ? void 0 : formControl$1.isInvalid;

  var _useBoolean = hooks.useBoolean(),
      isFocused = _useBoolean[0],
      setFocused = _useBoolean[1];

  var _useBoolean2 = hooks.useBoolean(),
      isHovered = _useBoolean2[0],
      setHovering = _useBoolean2[1];

  var _useBoolean3 = hooks.useBoolean(),
      isActive = _useBoolean3[0],
      setActive = _useBoolean3[1];

  var _useState = React.useState(Boolean(defaultChecked)),
      isCheckedState = _useState[0],
      setChecked = _useState[1];

  var _useControllableProp = hooks.useControllableProp(isCheckedProp, isCheckedState),
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
      setChecked(event.target.checked);
    }

    onChange == null ? void 0 : onChange(event);
  }, [isControlled, isDisabled, isReadOnly, onChange]);
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
  var getCheckboxProps = React.useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      "data-active": utils.dataAttr(isActive),
      "data-hover": utils.dataAttr(isHovered),
      "data-disabled": utils.dataAttr(isDisabled),
      "data-invalid": utils.dataAttr(isInvalid),
      "data-checked": utils.dataAttr(isChecked),
      "data-focus": utils.dataAttr(isFocused),
      "data-readonly": utils.dataAttr(isReadOnly),
      "aria-hidden": true,
      onMouseDown: utils.callAllHandlers(props.onMouseDown, setActive.on),
      onMouseUp: utils.callAllHandlers(props.onMouseUp, setActive.off),
      onMouseEnter: utils.callAllHandlers(props.onMouseEnter, setHovering.on),
      onMouseLeave: utils.callAllHandlers(props.onMouseLeave, setHovering.off)
    });
  }, [isActive, isHovered, isDisabled, isInvalid, isChecked, isFocused, isReadOnly, setActive.on, setActive.off, setHovering.on, setHovering.off]);

  var _ref = formControl$1 != null ? formControl$1 : {},
      onFocus = _ref.onFocus,
      onBlur = _ref.onBlur;

  var getInputProps = React.useCallback(function (props, ref) {
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
      onChange: utils.callAllHandlers(props.onChange, handleChange),
      onBlur: utils.callAllHandlers(onBlur, props.onBlur, setFocused.off),
      onFocus: utils.callAllHandlers(onFocus, props.onFocus, setFocused.on),
      onKeyDown: utils.callAllHandlers(props.onKeyDown, onKeyDown),
      onKeyUp: utils.callAllHandlers(props.onKeyUp, onKeyUp),
      checked: isChecked,
      disabled: trulyDisabled,
      readOnly: isReadOnly,
      required: isRequired,
      "aria-invalid": utils.ariaAttr(isInvalid),
      "aria-disabled": utils.ariaAttr(trulyDisabled),
      "aria-required": utils.ariaAttr(isRequired),
      "data-readonly": utils.dataAttr(isReadOnly),
      style: visuallyHidden.visuallyHiddenStyle
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
      onMouseDown: utils.callAllHandlers(props.onMouseDown, stop),
      onTouchStart: utils.callAllHandlers(props.onTouchStart, stop),
      "data-disabled": utils.dataAttr(isDisabled),
      "data-checked": utils.dataAttr(isChecked),
      "data-invalid": utils.dataAttr(isInvalid)
    });
  };

  var getRootProps = function getRootProps(props, ref) {
    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      "data-disabled": utils.dataAttr(isDisabled),
      "data-checked": utils.dataAttr(isChecked),
      "data-invalid": utils.dataAttr(isInvalid)
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
var Radio = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _props$name;

  var group = useRadioGroupContext();
  var onChangeProp = props.onChange,
      valueProp = props.value;
  var styles = system.useMultiStyleConfig("Radio", _extends({}, group, props));
  var ownProps = system.omitThemingProps(props);

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
    onChange = utils.callAll(group.onChange, onChangeProp);
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

  var _split = utils.split(htmlProps, system.layoutPropNames),
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

  return /*#__PURE__*/React__namespace.createElement(system.chakra.label, _extends({
    className: "chakra-radio"
  }, rootProps, {
    __css: rootStyles
  }), /*#__PURE__*/React__namespace.createElement("input", _extends({
    className: "chakra-radio__input"
  }, inputProps)), /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    className: "chakra-radio__control"
  }, checkboxProps, {
    __css: checkboxStyles
  })), children && /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    className: "chakra-radio__label"
  }, labelProps, {
    __css: labelStyles
  }), children));
});

if (utils.__DEV__) {
  Radio.displayName = "Radio";
}

exports.Radio = Radio;
exports.RadioGroup = RadioGroup;
exports.useRadio = useRadio;
exports.useRadioGroup = useRadioGroup;
exports.useRadioGroupContext = useRadioGroupContext;
