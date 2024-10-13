'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
var React = require('react');
var icon = require('@chakra-ui/icon');

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

var CheckIcon = function CheckIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    viewBox: "0 0 24 24"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M12,0A12,12,0,1,0,24,12,12.014,12.014,0,0,0,12,0Zm6.927,8.2-6.845,9.289a1.011,1.011,0,0,1-1.43.188L5.764,13.769a1,1,0,1,1,1.25-1.562l4.076,3.261,6.227-8.451A1,1,0,1,1,18.927,8.2Z"
  }));
};
var InfoIcon = function InfoIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    viewBox: "0 0 24 24"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M12,0A12,12,0,1,0,24,12,12.013,12.013,0,0,0,12,0Zm.25,5a1.5,1.5,0,1,1-1.5,1.5A1.5,1.5,0,0,1,12.25,5ZM14.5,18.5h-4a1,1,0,0,1,0-2h.75a.25.25,0,0,0,.25-.25v-4.5a.25.25,0,0,0-.25-.25H10.5a1,1,0,0,1,0-2h1a2,2,0,0,1,2,2v4.75a.25.25,0,0,0,.25.25h.75a1,1,0,1,1,0,2Z"
  }));
};
var WarningIcon = function WarningIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    viewBox: "0 0 24 24"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M11.983,0a12.206,12.206,0,0,0-8.51,3.653A11.8,11.8,0,0,0,0,12.207,11.779,11.779,0,0,0,11.8,24h.214A12.111,12.111,0,0,0,24,11.791h0A11.766,11.766,0,0,0,11.983,0ZM10.5,16.542a1.476,1.476,0,0,1,1.449-1.53h.027a1.527,1.527,0,0,1,1.523,1.47,1.475,1.475,0,0,1-1.449,1.53h-.027A1.529,1.529,0,0,1,10.5,16.542ZM11,12.5v-6a1,1,0,0,1,2,0v6a1,1,0,1,1-2,0Z"
  }));
};

var _excluded = ["status"];
var STATUSES = {
  info: {
    icon: InfoIcon,
    colorScheme: "blue"
  },
  warning: {
    icon: WarningIcon,
    colorScheme: "orange"
  },
  success: {
    icon: CheckIcon,
    colorScheme: "green"
  },
  error: {
    icon: WarningIcon,
    colorScheme: "red"
  }
};

var _createContext = reactUtils.createContext({
  name: "AlertContext",
  errorMessage: "useAlertContext: `context` is undefined. Seems you forgot to wrap alert components in `<Alert />`"
}),
    AlertProvider = _createContext[0],
    useAlertContext = _createContext[1];

/**
 * Alert is used to communicate the state or status of a
 * page, feature or action
 */
var Alert = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _props$colorScheme;

  var _omitThemingProps = system.omitThemingProps(props),
      _omitThemingProps$sta = _omitThemingProps.status,
      status = _omitThemingProps$sta === void 0 ? "info" : _omitThemingProps$sta,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var colorScheme = (_props$colorScheme = props.colorScheme) != null ? _props$colorScheme : STATUSES[status].colorScheme;
  var styles = system.useMultiStyleConfig("Alert", _extends({}, props, {
    colorScheme: colorScheme
  }));

  var alertStyles = _extends({
    width: "100%",
    display: "flex",
    alignItems: "center",
    position: "relative",
    overflow: "hidden"
  }, styles.container);

  return /*#__PURE__*/React__namespace.createElement(AlertProvider, {
    value: {
      status: status
    }
  }, /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    role: "alert",
    ref: ref
  }, rest, {
    className: utils.cx("chakra-alert", props.className),
    __css: alertStyles
  }))));
});
var AlertTitle = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    ref: ref
  }, props, {
    className: utils.cx("chakra-alert__title", props.className),
    __css: styles.title
  }));
});
var AlertDescription = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();

  var descriptionStyles = _extends({
    display: "inline"
  }, styles.description);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    ref: ref
  }, props, {
    className: utils.cx("chakra-alert__desc", props.className),
    __css: descriptionStyles
  }));
});
var AlertIcon = function AlertIcon(props) {
  var _useAlertContext = useAlertContext(),
      status = _useAlertContext.status;

  var BaseIcon = STATUSES[status].icon;
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    display: "inherit"
  }, props, {
    className: utils.cx("chakra-alert__icon", props.className),
    __css: styles.icon
  }), /*#__PURE__*/React__namespace.createElement(BaseIcon, {
    w: "100%",
    h: "100%"
  }));
};

exports.Alert = Alert;
exports.AlertDescription = AlertDescription;
exports.AlertIcon = AlertIcon;
exports.AlertTitle = AlertTitle;
