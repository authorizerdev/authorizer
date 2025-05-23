import _extends from "@babel/runtime/helpers/esm/extends";
import React, { forwardRef, useRef, useState, useCallback, useEffect, useMemo, Fragment } from 'react';
import { node, bool, string, any, arrayOf, oneOfType, object, func } from 'prop-types';
import { FOCUS_DISABLED, FOCUS_GROUP } from 'focus-lock/constants';
import { useMergeRefs } from 'use-callback-ref';
import { hiddenGuard } from './FocusGuard';
import { mediumFocus, mediumBlur, mediumSidecar } from './medium';
import { focusScope } from './scope';
var emptyArray = [];
var FocusLock = /*#__PURE__*/forwardRef(function FocusLockUI(props, parentRef) {
  var _extends2;
  var _useState = useState(),
    realObserved = _useState[0],
    setObserved = _useState[1];
  var observed = useRef();
  var isActive = useRef(false);
  var originalFocusedElement = useRef(null);
  var _useState2 = useState({}),
    update = _useState2[1];
  var children = props.children,
    _props$disabled = props.disabled,
    disabled = _props$disabled === void 0 ? false : _props$disabled,
    _props$noFocusGuards = props.noFocusGuards,
    noFocusGuards = _props$noFocusGuards === void 0 ? false : _props$noFocusGuards,
    _props$persistentFocu = props.persistentFocus,
    persistentFocus = _props$persistentFocu === void 0 ? false : _props$persistentFocu,
    _props$crossFrame = props.crossFrame,
    crossFrame = _props$crossFrame === void 0 ? true : _props$crossFrame,
    _props$autoFocus = props.autoFocus,
    autoFocus = _props$autoFocus === void 0 ? true : _props$autoFocus,
    allowTextSelection = props.allowTextSelection,
    group = props.group,
    className = props.className,
    whiteList = props.whiteList,
    hasPositiveIndices = props.hasPositiveIndices,
    _props$shards = props.shards,
    shards = _props$shards === void 0 ? emptyArray : _props$shards,
    _props$as = props.as,
    Container = _props$as === void 0 ? 'div' : _props$as,
    _props$lockProps = props.lockProps,
    containerProps = _props$lockProps === void 0 ? {} : _props$lockProps,
    SideCar = props.sideCar,
    _props$returnFocus = props.returnFocus,
    shouldReturnFocus = _props$returnFocus === void 0 ? false : _props$returnFocus,
    focusOptions = props.focusOptions,
    onActivationCallback = props.onActivation,
    onDeactivationCallback = props.onDeactivation;
  var _useState3 = useState({}),
    id = _useState3[0];
  var onActivation = useCallback(function (_ref) {
    var captureFocusRestore = _ref.captureFocusRestore;
    if (!originalFocusedElement.current) {
      var _document;
      var activeElement = (_document = document) == null ? void 0 : _document.activeElement;
      originalFocusedElement.current = activeElement;
      if (activeElement !== document.body) {
        originalFocusedElement.current = captureFocusRestore(activeElement);
      }
    }
    if (observed.current && onActivationCallback) {
      onActivationCallback(observed.current);
    }
    isActive.current = true;
    update();
  }, [onActivationCallback]);
  var onDeactivation = useCallback(function () {
    isActive.current = false;
    if (onDeactivationCallback) {
      onDeactivationCallback(observed.current);
    }
    update();
  }, [onDeactivationCallback]);
  var returnFocus = useCallback(function (allowDefer) {
    var focusRestore = originalFocusedElement.current;
    if (focusRestore) {
      var returnFocusTo = (typeof focusRestore === 'function' ? focusRestore() : focusRestore) || document.body;
      var howToReturnFocus = typeof shouldReturnFocus === 'function' ? shouldReturnFocus(returnFocusTo) : shouldReturnFocus;
      if (howToReturnFocus) {
        var returnFocusOptions = typeof howToReturnFocus === 'object' ? howToReturnFocus : undefined;
        originalFocusedElement.current = null;
        if (allowDefer) {
          Promise.resolve().then(function () {
            return returnFocusTo.focus(returnFocusOptions);
          });
        } else {
          returnFocusTo.focus(returnFocusOptions);
        }
      }
    }
  }, [shouldReturnFocus]);
  var onFocus = useCallback(function (event) {
    if (isActive.current) {
      mediumFocus.useMedium(event);
    }
  }, []);
  var onBlur = mediumBlur.useMedium;
  var setObserveNode = useCallback(function (newObserved) {
    if (observed.current !== newObserved) {
      observed.current = newObserved;
      setObserved(newObserved);
    }
  }, []);
  if (process.env.NODE_ENV !== 'production') {
    if (typeof allowTextSelection !== 'undefined') {
      console.warn('React-Focus-Lock: allowTextSelection is deprecated and enabled by default');
    }
    useEffect(function () {
      if (!observed.current && typeof Container !== 'string') {
        console.error('FocusLock: could not obtain ref to internal node');
      }
    }, []);
  }
  var lockProps = _extends((_extends2 = {}, _extends2[FOCUS_DISABLED] = disabled && 'disabled', _extends2[FOCUS_GROUP] = group, _extends2), containerProps);
  var hasLeadingGuards = noFocusGuards !== true;
  var hasTailingGuards = hasLeadingGuards && noFocusGuards !== 'tail';
  var mergedRef = useMergeRefs([parentRef, setObserveNode]);
  var focusScopeValue = useMemo(function () {
    return {
      observed: observed,
      shards: shards,
      enabled: !disabled,
      active: isActive.current
    };
  }, [disabled, isActive.current, shards, realObserved]);
  return /*#__PURE__*/React.createElement(Fragment, null, hasLeadingGuards && [
  /*#__PURE__*/
  React.createElement("div", {
    key: "guard-first",
    "data-focus-guard": true,
    tabIndex: disabled ? -1 : 0,
    style: hiddenGuard
  }), hasPositiveIndices ? /*#__PURE__*/React.createElement("div", {
    key: "guard-nearest",
    "data-focus-guard": true,
    tabIndex: disabled ? -1 : 1,
    style: hiddenGuard
  }) : null], !disabled && /*#__PURE__*/React.createElement(SideCar, {
    id: id,
    sideCar: mediumSidecar,
    observed: realObserved,
    disabled: disabled,
    persistentFocus: persistentFocus,
    crossFrame: crossFrame,
    autoFocus: autoFocus,
    whiteList: whiteList,
    shards: shards,
    onActivation: onActivation,
    onDeactivation: onDeactivation,
    returnFocus: returnFocus,
    focusOptions: focusOptions,
    noFocusGuards: noFocusGuards
  }), /*#__PURE__*/React.createElement(Container, _extends({
    ref: mergedRef
  }, lockProps, {
    className: className,
    onBlur: onBlur,
    onFocus: onFocus
  }), /*#__PURE__*/React.createElement(focusScope.Provider, {
    value: focusScopeValue
  }, children)), hasTailingGuards && /*#__PURE__*/React.createElement("div", {
    "data-focus-guard": true,
    tabIndex: disabled ? -1 : 0,
    style: hiddenGuard
  }));
});
FocusLock.propTypes = process.env.NODE_ENV !== "production" ? {
  children: node,
  disabled: bool,
  returnFocus: oneOfType([bool, object, func]),
  focusOptions: object,
  noFocusGuards: bool,
  hasPositiveIndices: bool,
  allowTextSelection: bool,
  autoFocus: bool,
  persistentFocus: bool,
  crossFrame: bool,
  group: string,
  className: string,
  whiteList: func,
  shards: arrayOf(any),
  as: oneOfType([string, func, object]),
  lockProps: object,
  onActivation: func,
  onDeactivation: func,
  sideCar: any.isRequired
} : {};
export default FocusLock;