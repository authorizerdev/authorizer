'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var hooks = require('@chakra-ui/hooks');
var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
var React = require('react');
var Icon = require('@chakra-ui/icon');

function _interopDefault (e) { return e && e.__esModule ? e : { 'default': e }; }

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
var Icon__default = /*#__PURE__*/_interopDefault(Icon);

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

var _excluded$2 = ["id", "isRequired", "isInvalid", "isDisabled", "isReadOnly"],
    _excluded2$1 = ["getRootProps", "htmlProps"];

var _createContext = reactUtils.createContext({
  strict: false,
  name: "FormControlContext"
}),
    FormControlProvider = _createContext[0],
    useFormControlContext = _createContext[1];

function useFormControlProvider(props) {
  var idProp = props.id,
      isRequired = props.isRequired,
      isInvalid = props.isInvalid,
      isDisabled = props.isDisabled,
      isReadOnly = props.isReadOnly,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded$2); // Generate all the required ids


  var uuid = hooks.useId();
  var id = idProp || "field-" + uuid;
  var labelId = id + "-label";
  var feedbackId = id + "-feedback";
  var helpTextId = id + "-helptext";
  /**
   * Track whether the `FormErrorMessage` has been rendered.
   * We use this to append its id the the `aria-describedby` of the `input`.
   */

  var _React$useState = React__namespace.useState(false),
      hasFeedbackText = _React$useState[0],
      setHasFeedbackText = _React$useState[1];
  /**
   * Track whether the `FormHelperText` has been rendered.
   * We use this to append its id the the `aria-describedby` of the `input`.
   */


  var _React$useState2 = React__namespace.useState(false),
      hasHelpText = _React$useState2[0],
      setHasHelpText = _React$useState2[1]; // Track whether the form element (e.g, `input`) has focus.


  var _useBoolean = hooks.useBoolean(),
      isFocused = _useBoolean[0],
      setFocus = _useBoolean[1];

  var getHelpTextProps = React__namespace.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({
      id: helpTextId
    }, props, {
      /**
       * Notify the field context when the help text is rendered on screen,
       * so we can apply the correct `aria-describedby` to the field (e.g. input, textarea).
       */
      ref: reactUtils.mergeRefs(forwardedRef, function (node) {
        if (!node) return;
        setHasHelpText(true);
      })
    });
  }, [helpTextId]);
  var getLabelProps = React__namespace.useCallback(function (props, forwardedRef) {
    var _props$id, _props$htmlFor;

    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: forwardedRef,
      "data-focus": utils.dataAttr(isFocused),
      "data-disabled": utils.dataAttr(isDisabled),
      "data-invalid": utils.dataAttr(isInvalid),
      "data-readonly": utils.dataAttr(isReadOnly),
      id: (_props$id = props.id) != null ? _props$id : labelId,
      htmlFor: (_props$htmlFor = props.htmlFor) != null ? _props$htmlFor : id
    });
  }, [id, isDisabled, isFocused, isInvalid, isReadOnly, labelId]);
  var getErrorMessageProps = React__namespace.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({
      id: feedbackId
    }, props, {
      /**
       * Notify the field context when the error message is rendered on screen,
       * so we can apply the correct `aria-describedby` to the field (e.g. input, textarea).
       */
      ref: reactUtils.mergeRefs(forwardedRef, function (node) {
        if (!node) return;
        setHasFeedbackText(true);
      }),
      "aria-live": "polite"
    });
  }, [feedbackId]);
  var getRootProps = React__namespace.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, htmlProps, {
      ref: forwardedRef,
      role: "group"
    });
  }, [htmlProps]);
  var getRequiredIndicatorProps = React__namespace.useCallback(function (props, forwardedRef) {
    if (props === void 0) {
      props = {};
    }

    if (forwardedRef === void 0) {
      forwardedRef = null;
    }

    return _extends({}, props, {
      ref: forwardedRef,
      role: "presentation",
      "aria-hidden": true,
      children: props.children || "*"
    });
  }, []);
  return {
    isRequired: !!isRequired,
    isInvalid: !!isInvalid,
    isReadOnly: !!isReadOnly,
    isDisabled: !!isDisabled,
    isFocused: !!isFocused,
    onFocus: setFocus.on,
    onBlur: setFocus.off,
    hasFeedbackText: hasFeedbackText,
    setHasFeedbackText: setHasFeedbackText,
    hasHelpText: hasHelpText,
    setHasHelpText: setHasHelpText,
    id: id,
    labelId: labelId,
    feedbackId: feedbackId,
    helpTextId: helpTextId,
    htmlProps: htmlProps,
    getHelpTextProps: getHelpTextProps,
    getErrorMessageProps: getErrorMessageProps,
    getRootProps: getRootProps,
    getLabelProps: getLabelProps,
    getRequiredIndicatorProps: getRequiredIndicatorProps
  };
}

/**
 * FormControl provides context such as
 * `isInvalid`, `isDisabled`, and `isRequired` to form elements.
 *
 * This is commonly used in form elements such as `input`,
 * `select`, `textarea`, etc.
 */
var FormControl = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("Form", props);
  var ownProps = system.omitThemingProps(props);

  var _useFormControlProvid = useFormControlProvider(ownProps),
      getRootProps = _useFormControlProvid.getRootProps;
      _useFormControlProvid.htmlProps;
      var context = _objectWithoutPropertiesLoose(_useFormControlProvid, _excluded2$1);

  var className = utils.cx("chakra-form-control", props.className);
  var contextValue = React__namespace.useMemo(function () {
    return context;
  }, [context]);
  return /*#__PURE__*/React__namespace.createElement(FormControlProvider, {
    value: contextValue
  }, /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, getRootProps({}, ref), {
    className: className,
    __css: styles["container"]
  }))));
});

if (utils.__DEV__) {
  FormControl.displayName = "FormControl";
}

/**
 * FormHelperText
 *
 * Assistive component that conveys additional guidance
 * about the field, such as how it will be used and what
 * types in values should be provided.
 */
var FormHelperText = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var field = useFormControlContext();
  var styles = system.useStyles();
  var className = utils.cx("chakra-form__helper-text", props.className);
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, field == null ? void 0 : field.getHelpTextProps(props, ref), {
    __css: styles.helperText,
    className: className
  }));
});

if (utils.__DEV__) {
  FormHelperText.displayName = "FormHelperText";
}

var _excluded$1 = ["isDisabled", "isInvalid", "isReadOnly", "isRequired"],
    _excluded2 = ["id", "disabled", "readOnly", "required", "isRequired", "isInvalid", "isReadOnly", "isDisabled", "onFocus", "onBlur"];

/**
 * React hook that provides the props that should be spread on to
 * input fields (`input`, `select`, `textarea`, etc.).
 *
 * It provides a convenient way to control a form fields, validation
 * and helper text.
 *
 * @internal
 */
function useFormControl(props) {
  var _useFormControlProps = useFormControlProps(props),
      isDisabled = _useFormControlProps.isDisabled,
      isInvalid = _useFormControlProps.isInvalid,
      isReadOnly = _useFormControlProps.isReadOnly,
      isRequired = _useFormControlProps.isRequired,
      rest = _objectWithoutPropertiesLoose(_useFormControlProps, _excluded$1);

  return _extends({}, rest, {
    disabled: isDisabled,
    readOnly: isReadOnly,
    required: isRequired,
    "aria-invalid": utils.ariaAttr(isInvalid),
    "aria-required": utils.ariaAttr(isRequired),
    "aria-readonly": utils.ariaAttr(isReadOnly)
  });
}
/**
 * @internal
 */

function useFormControlProps(props) {
  var _ref, _ref2, _ref3;

  var field = useFormControlContext();

  var id = props.id,
      disabled = props.disabled,
      readOnly = props.readOnly,
      required = props.required,
      isRequired = props.isRequired,
      isInvalid = props.isInvalid,
      isReadOnly = props.isReadOnly,
      isDisabled = props.isDisabled,
      onFocus = props.onFocus,
      onBlur = props.onBlur,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);

  var labelIds = props["aria-describedby"] ? [props["aria-describedby"]] : []; // Error message must be described first in all scenarios.

  if (field != null && field.hasFeedbackText && field != null && field.isInvalid) {
    labelIds.push(field.feedbackId);
  }

  if (field != null && field.hasHelpText) {
    labelIds.push(field.helpTextId);
  }

  return _extends({}, rest, {
    "aria-describedby": labelIds.join(" ") || undefined,
    id: id != null ? id : field == null ? void 0 : field.id,
    isDisabled: (_ref = disabled != null ? disabled : isDisabled) != null ? _ref : field == null ? void 0 : field.isDisabled,
    isReadOnly: (_ref2 = readOnly != null ? readOnly : isReadOnly) != null ? _ref2 : field == null ? void 0 : field.isReadOnly,
    isRequired: (_ref3 = required != null ? required : isRequired) != null ? _ref3 : field == null ? void 0 : field.isRequired,
    isInvalid: isInvalid != null ? isInvalid : field == null ? void 0 : field.isInvalid,
    onFocus: utils.callAllHandlers(field == null ? void 0 : field.onFocus, onFocus),
    onBlur: utils.callAllHandlers(field == null ? void 0 : field.onBlur, onBlur)
  });
}

/**
 * Used to provide feedback about an invalid input,
 * and suggest clear instructions on how to fix it.
 */
var FormErrorMessage = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("FormError", props);
  var ownProps = system.omitThemingProps(props);
  var field = useFormControlContext();
  if (!(field != null && field.isInvalid)) return null;
  return /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, field == null ? void 0 : field.getErrorMessageProps(ownProps, ref), {
    className: utils.cx("chakra-form__error-message", props.className),
    __css: _extends({
      display: "flex",
      alignItems: "center"
    }, styles.text)
  })));
});

if (utils.__DEV__) {
  FormErrorMessage.displayName = "FormErrorMessage";
}
/**
 * Used as the visual indicator that a field is invalid or
 * a field has incorrect values.
 */


var FormErrorIcon = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  var field = useFormControlContext();
  if (!(field != null && field.isInvalid)) return null;

  var _className = utils.cx("chakra-form__error-icon", props.className);

  return /*#__PURE__*/React__namespace.createElement(Icon__default["default"], _extends({
    ref: ref,
    "aria-hidden": true
  }, props, {
    __css: styles.icon,
    className: _className
  }), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M11.983,0a12.206,12.206,0,0,0-8.51,3.653A11.8,11.8,0,0,0,0,12.207,11.779,11.779,0,0,0,11.8,24h.214A12.111,12.111,0,0,0,24,11.791h0A11.766,11.766,0,0,0,11.983,0ZM10.5,16.542a1.476,1.476,0,0,1,1.449-1.53h.027a1.527,1.527,0,0,1,1.523,1.47,1.475,1.475,0,0,1-1.449,1.53h-.027A1.529,1.529,0,0,1,10.5,16.542ZM11,12.5v-6a1,1,0,0,1,2,0v6a1,1,0,1,1-2,0Z"
  }));
});

if (utils.__DEV__) {
  FormErrorIcon.displayName = "FormErrorIcon";
}

var _excluded = ["className", "children", "requiredIndicator"];

/**
 * Used to enhance the usability of form controls.
 *
 * It is used to inform users as to what information
 * is requested for a form field.
 *
 * ♿️ Accessibility: Every form field should have a form label.
 */
var FormLabel = /*#__PURE__*/system.forwardRef(function (passedProps, ref) {
  var _field$getLabelProps;

  var styles = system.useStyleConfig("FormLabel", passedProps);
  var props = system.omitThemingProps(passedProps);

  props.className;
      var children = props.children,
      _props$requiredIndica = props.requiredIndicator,
      requiredIndicator = _props$requiredIndica === void 0 ? /*#__PURE__*/React__namespace.createElement(RequiredIndicator, null) : _props$requiredIndica,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var field = useFormControlContext();
  var ownProps = (_field$getLabelProps = field == null ? void 0 : field.getLabelProps(rest, ref)) != null ? _field$getLabelProps : _extends({
    ref: ref
  }, rest);
  return /*#__PURE__*/React__namespace.createElement(system.chakra.label, _extends({}, ownProps, {
    className: utils.cx("chakra-form__label", props.className),
    __css: _extends({
      display: "block",
      textAlign: "start"
    }, styles)
  }), children, field != null && field.isRequired ? requiredIndicator : null);
});

if (utils.__DEV__) {
  FormLabel.displayName = "FormLabel";
}

/**
 * Used to show a "required" text or an asterisks (*) to indicate that
 * a field is required.
 */
var RequiredIndicator = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var field = useFormControlContext();
  var styles = system.useStyles();
  if (!(field != null && field.isRequired)) return null;
  var className = utils.cx("chakra-form__required-indicator", props.className);
  return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({}, field == null ? void 0 : field.getRequiredIndicatorProps(props, ref), {
    __css: styles.requiredIndicator,
    className: className
  }));
});

if (utils.__DEV__) {
  RequiredIndicator.displayName = "RequiredIndicator";
}

exports.FormControl = FormControl;
exports.FormErrorIcon = FormErrorIcon;
exports.FormErrorMessage = FormErrorMessage;
exports.FormHelperText = FormHelperText;
exports.FormLabel = FormLabel;
exports.RequiredIndicator = RequiredIndicator;
exports.useFormControl = useFormControl;
exports.useFormControlContext = useFormControlContext;
exports.useFormControlProps = useFormControlProps;
