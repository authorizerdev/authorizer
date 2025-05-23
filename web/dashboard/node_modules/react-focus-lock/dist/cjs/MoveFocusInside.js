"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");
var _typeof = require("@babel/runtime/helpers/typeof");
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.useFocusInside = exports["default"] = void 0;
var _extends2 = _interopRequireDefault(require("@babel/runtime/helpers/extends"));
var _react = _interopRequireWildcard(require("react"));
var _propTypes = _interopRequireDefault(require("prop-types"));
var _constants = require("focus-lock/constants");
var _util = require("./util");
var _medium = require("./medium");
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function _getRequireWildcardCache(e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != _typeof(e) && "function" != typeof e) return { "default": e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && Object.prototype.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n["default"] = e, t && t.set(e, n), n; }
var useFocusInside = exports.useFocusInside = function useFocusInside(observedRef) {
  (0, _react.useEffect)(function () {
    var enabled = true;
    _medium.mediumEffect.useMedium(function (car) {
      var observed = observedRef && observedRef.current;
      if (enabled && observed) {
        if (!car.focusInside(observed)) {
          car.moveFocusInside(observed, null);
        }
      }
    });
    return function () {
      enabled = false;
    };
  }, [observedRef]);
};
function MoveFocusInside(_ref) {
  var _ref$disabled = _ref.disabled,
    isDisabled = _ref$disabled === void 0 ? false : _ref$disabled,
    className = _ref.className,
    children = _ref.children;
  var ref = (0, _react.useRef)(null);
  useFocusInside(isDisabled ? undefined : ref);
  return /*#__PURE__*/_react["default"].createElement("div", (0, _extends2["default"])({}, (0, _util.inlineProp)(_constants.FOCUS_AUTO, !isDisabled), {
    ref: ref,
    className: className
  }), children);
}
MoveFocusInside.propTypes = process.env.NODE_ENV !== "production" ? {
  children: _propTypes["default"].node.isRequired,
  disabled: _propTypes["default"].bool,
  className: _propTypes["default"].string
} : {};
var _default = exports["default"] = MoveFocusInside;