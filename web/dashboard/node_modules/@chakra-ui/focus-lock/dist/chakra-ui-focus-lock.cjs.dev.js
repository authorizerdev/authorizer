'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var React = require('react');
var ReactFocusLock = require('react-focus-lock');
var utils = require('@chakra-ui/utils');

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
var ReactFocusLock__default = /*#__PURE__*/_interopDefault(ReactFocusLock);

var FocusLock = function FocusLock(props) {
  var initialFocusRef = props.initialFocusRef,
      finalFocusRef = props.finalFocusRef,
      contentRef = props.contentRef,
      restoreFocus = props.restoreFocus,
      children = props.children,
      isDisabled = props.isDisabled,
      autoFocus = props.autoFocus,
      persistentFocus = props.persistentFocus,
      lockFocusAcrossFrames = props.lockFocusAcrossFrames;
  var onActivation = React__namespace.useCallback(function () {
    if (initialFocusRef != null && initialFocusRef.current) {
      initialFocusRef.current.focus();
    } else if (contentRef != null && contentRef.current) {
      var focusables = utils.getAllFocusable(contentRef.current);

      if (focusables.length === 0) {
        utils.focus(contentRef.current, {
          nextTick: true
        });
      }
    }
  }, [initialFocusRef, contentRef]);
  var onDeactivation = React__namespace.useCallback(function () {
    var _finalFocusRef$curren;

    finalFocusRef == null ? void 0 : (_finalFocusRef$curren = finalFocusRef.current) == null ? void 0 : _finalFocusRef$curren.focus();
  }, [finalFocusRef]);
  var returnFocus = restoreFocus && !finalFocusRef;
  return /*#__PURE__*/React__namespace.createElement(ReactFocusLock__default["default"], {
    crossFrame: lockFocusAcrossFrames,
    persistentFocus: persistentFocus,
    autoFocus: autoFocus,
    disabled: isDisabled,
    onActivation: onActivation,
    onDeactivation: onDeactivation,
    returnFocus: returnFocus
  }, children);
};

if (utils.__DEV__) {
  FocusLock.displayName = "FocusLock";
}

exports.FocusLock = FocusLock;
exports["default"] = FocusLock;
