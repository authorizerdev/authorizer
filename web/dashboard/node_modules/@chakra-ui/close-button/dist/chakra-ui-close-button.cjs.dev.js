'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var icon = require('@chakra-ui/icon');
var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var React = require('react');

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

var _excluded = ["children", "isDisabled", "__css"];

var CloseIcon = function CloseIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    focusable: "false",
    "aria-hidden": true
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M.439,21.44a1.5,1.5,0,0,0,2.122,2.121L11.823,14.3a.25.25,0,0,1,.354,0l9.262,9.263a1.5,1.5,0,1,0,2.122-2.121L14.3,12.177a.25.25,0,0,1,0-.354l9.263-9.262A1.5,1.5,0,0,0,21.439.44L12.177,9.7a.25.25,0,0,1-.354,0L2.561.44A1.5,1.5,0,0,0,.439,2.561L9.7,11.823a.25.25,0,0,1,0,.354Z"
  }));
};

/**
 * A button with a close icon.
 *
 * It is used to handle the close functionality in feedback and overlay components
 * like Alerts, Toasts, Drawers and Modals.
 */
var CloseButton = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyleConfig("CloseButton", props);

  var _omitThemingProps = system.omitThemingProps(props),
      children = _omitThemingProps.children,
      isDisabled = _omitThemingProps.isDisabled,
      __css = _omitThemingProps.__css,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var baseStyle = {
    outline: 0,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0
  };
  return /*#__PURE__*/React__namespace.createElement(system.chakra.button, _extends({
    type: "button",
    "aria-label": "Close",
    ref: ref,
    disabled: isDisabled,
    __css: _extends({}, baseStyle, styles, __css)
  }, rest), children || /*#__PURE__*/React__namespace.createElement(CloseIcon, {
    width: "1em",
    height: "1em"
  }));
});

if (utils.__DEV__) {
  CloseButton.displayName = "CloseButton";
}

exports.CloseButton = CloseButton;
