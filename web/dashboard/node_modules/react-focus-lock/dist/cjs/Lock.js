"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");
var _typeof3 = require("@babel/runtime/helpers/typeof");
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports["default"] = void 0;
var _extends2 = _interopRequireDefault(require("@babel/runtime/helpers/extends"));
var _defineProperty2 = _interopRequireDefault(require("@babel/runtime/helpers/defineProperty"));
var _typeof2 = _interopRequireDefault(require("@babel/runtime/helpers/typeof"));
var _slicedToArray2 = _interopRequireDefault(require("@babel/runtime/helpers/slicedToArray"));
var _react = _interopRequireWildcard(require("react"));
var _propTypes = require("prop-types");
var _constants = require("focus-lock/constants");
var _useCallbackRef = require("use-callback-ref");
var _FocusGuard = require("./FocusGuard");
var _medium = require("./medium");
var _scope = require("./scope");
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function _getRequireWildcardCache(e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != _typeof3(e) && "function" != typeof e) return { "default": e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && Object.prototype.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n["default"] = e, t && t.set(e, n), n; }
function ownKeys(e, r) { var t = Object.keys(e); if (Object.getOwnPropertySymbols) { var o = Object.getOwnPropertySymbols(e); r && (o = o.filter(function (r) { return Object.getOwnPropertyDescriptor(e, r).enumerable; })), t.push.apply(t, o); } return t; }
function _objectSpread(e) { for (var r = 1; r < arguments.length; r++) { var t = null != arguments[r] ? arguments[r] : {}; r % 2 ? ownKeys(Object(t), !0).forEach(function (r) { (0, _defineProperty2["default"])(e, r, t[r]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(e, Object.getOwnPropertyDescriptors(t)) : ownKeys(Object(t)).forEach(function (r) { Object.defineProperty(e, r, Object.getOwnPropertyDescriptor(t, r)); }); } return e; }
var emptyArray = [];
var FocusLock = /*#__PURE__*/(0, _react.forwardRef)(function FocusLockUI(props, parentRef) {
  var _useState = (0, _react.useState)(),
    _useState2 = (0, _slicedToArray2["default"])(_useState, 2),
    realObserved = _useState2[0],
    setObserved = _useState2[1];
  var observed = (0, _react.useRef)();
  var isActive = (0, _react.useRef)(false);
  var originalFocusedElement = (0, _react.useRef)(null);
  var _useState3 = (0, _react.useState)({}),
    _useState4 = (0, _slicedToArray2["default"])(_useState3, 2),
    update = _useState4[1];
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
  var _useState5 = (0, _react.useState)({}),
    _useState6 = (0, _slicedToArray2["default"])(_useState5, 1),
    id = _useState6[0];
  var onActivation = (0, _react.useCallback)(function (_ref) {
    var captureFocusRestore = _ref.captureFocusRestore;
    if (!originalFocusedElement.current) {
      var _document;
      var activeElement = (_document = document) === null || _document === void 0 ? void 0 : _document.activeElement;
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
  var onDeactivation = (0, _react.useCallback)(function () {
    isActive.current = false;
    if (onDeactivationCallback) {
      onDeactivationCallback(observed.current);
    }
    update();
  }, [onDeactivationCallback]);
  var returnFocus = (0, _react.useCallback)(function (allowDefer) {
    var focusRestore = originalFocusedElement.current;
    if (focusRestore) {
      var returnFocusTo = (typeof focusRestore === 'function' ? focusRestore() : focusRestore) || document.body;
      var howToReturnFocus = typeof shouldReturnFocus === 'function' ? shouldReturnFocus(returnFocusTo) : shouldReturnFocus;
      if (howToReturnFocus) {
        var returnFocusOptions = (0, _typeof2["default"])(howToReturnFocus) === 'object' ? howToReturnFocus : undefined;
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
  var onFocus = (0, _react.useCallback)(function (event) {
    if (isActive.current) {
      _medium.mediumFocus.useMedium(event);
    }
  }, []);
  var onBlur = _medium.mediumBlur.useMedium;
  var setObserveNode = (0, _react.useCallback)(function (newObserved) {
    if (observed.current !== newObserved) {
      observed.current = newObserved;
      setObserved(newObserved);
    }
  }, []);
  if (process.env.NODE_ENV !== 'production') {
    if (typeof allowTextSelection !== 'undefined') {
      console.warn('React-Focus-Lock: allowTextSelection is deprecated and enabled by default');
    }
    (0, _react.useEffect)(function () {
      if (!observed.current && typeof Container !== 'string') {
        console.error('FocusLock: could not obtain ref to internal node');
      }
    }, []);
  }
  var lockProps = _objectSpread((0, _defineProperty2["default"])((0, _defineProperty2["default"])({}, _constants.FOCUS_DISABLED, disabled && 'disabled'), _constants.FOCUS_GROUP, group), containerProps);
  var hasLeadingGuards = noFocusGuards !== true;
  var hasTailingGuards = hasLeadingGuards && noFocusGuards !== 'tail';
  var mergedRef = (0, _useCallbackRef.useMergeRefs)([parentRef, setObserveNode]);
  var focusScopeValue = (0, _react.useMemo)(function () {
    return {
      observed: observed,
      shards: shards,
      enabled: !disabled,
      active: isActive.current
    };
  }, [disabled, isActive.current, shards, realObserved]);
  return /*#__PURE__*/_react["default"].createElement(_react.Fragment, null, hasLeadingGuards && [
  /*#__PURE__*/
  _react["default"].createElement("div", {
    key: "guard-first",
    "data-focus-guard": true,
    tabIndex: disabled ? -1 : 0,
    style: _FocusGuard.hiddenGuard
  }), hasPositiveIndices ? /*#__PURE__*/_react["default"].createElement("div", {
    key: "guard-nearest",
    "data-focus-guard": true,
    tabIndex: disabled ? -1 : 1,
    style: _FocusGuard.hiddenGuard
  }) : null], !disabled && /*#__PURE__*/_react["default"].createElement(SideCar, {
    id: id,
    sideCar: _medium.mediumSidecar,
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
  }), /*#__PURE__*/_react["default"].createElement(Container, (0, _extends2["default"])({
    ref: mergedRef
  }, lockProps, {
    className: className,
    onBlur: onBlur,
    onFocus: onFocus
  }), /*#__PURE__*/_react["default"].createElement(_scope.focusScope.Provider, {
    value: focusScopeValue
  }, children)), hasTailingGuards && /*#__PURE__*/_react["default"].createElement("div", {
    "data-focus-guard": true,
    tabIndex: disabled ? -1 : 0,
    style: _FocusGuard.hiddenGuard
  }));
});
FocusLock.propTypes = process.env.NODE_ENV !== "production" ? {
  children: _propTypes.node,
  disabled: _propTypes.bool,
  returnFocus: (0, _propTypes.oneOfType)([_propTypes.bool, _propTypes.object, _propTypes.func]),
  focusOptions: _propTypes.object,
  noFocusGuards: _propTypes.bool,
  hasPositiveIndices: _propTypes.bool,
  allowTextSelection: _propTypes.bool,
  autoFocus: _propTypes.bool,
  persistentFocus: _propTypes.bool,
  crossFrame: _propTypes.bool,
  group: _propTypes.string,
  className: _propTypes.string,
  whiteList: _propTypes.func,
  shards: (0, _propTypes.arrayOf)(_propTypes.any),
  as: (0, _propTypes.oneOfType)([_propTypes.string, _propTypes.func, _propTypes.object]),
  lockProps: _propTypes.object,
  onActivation: _propTypes.func,
  onDeactivation: _propTypes.func,
  sideCar: _propTypes.any.isRequired
} : {};
var _default = exports["default"] = FocusLock;