'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var CSSReset = require('@chakra-ui/css-reset');
var portal = require('@chakra-ui/portal');
var system = require('@chakra-ui/system');
var reactEnv = require('@chakra-ui/react-env');
var React = require('react');
var hooks = require('@chakra-ui/hooks');

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

var CSSReset__default = /*#__PURE__*/_interopDefault(CSSReset);
var React__namespace = /*#__PURE__*/_interopNamespace(React);

/**
 * The global provider that must be added to make all Chakra components
 * work correctly
 */
var ChakraProvider = function ChakraProvider(props) {
  var children = props.children,
      colorModeManager = props.colorModeManager,
      portalZIndex = props.portalZIndex,
      _props$resetCSS = props.resetCSS,
      resetCSS = _props$resetCSS === void 0 ? true : _props$resetCSS,
      _props$theme = props.theme,
      theme = _props$theme === void 0 ? {} : _props$theme,
      environment = props.environment,
      cssVarsRoot = props.cssVarsRoot;

  var _children = /*#__PURE__*/React__namespace.createElement(reactEnv.EnvironmentProvider, {
    environment: environment
  }, children);

  return /*#__PURE__*/React__namespace.createElement(hooks.IdProvider, null, /*#__PURE__*/React__namespace.createElement(system.ThemeProvider, {
    theme: theme,
    cssVarsRoot: cssVarsRoot
  }, /*#__PURE__*/React__namespace.createElement(system.ColorModeProvider, {
    colorModeManager: colorModeManager,
    options: theme.config
  }, resetCSS && /*#__PURE__*/React__namespace.createElement(CSSReset__default["default"], null), /*#__PURE__*/React__namespace.createElement(system.GlobalStyle, null), portalZIndex ? /*#__PURE__*/React__namespace.createElement(portal.PortalManager, {
    zIndex: portalZIndex
  }, _children) : _children)));
};

exports.ChakraProvider = ChakraProvider;
