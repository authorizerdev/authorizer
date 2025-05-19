"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");
var _typeof = require("@babel/runtime/helpers/typeof");
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports["default"] = void 0;
var _objectWithoutProperties2 = _interopRequireDefault(require("@babel/runtime/helpers/objectWithoutProperties"));
var _extends2 = _interopRequireDefault(require("@babel/runtime/helpers/extends"));
var _react = _interopRequireWildcard(require("react"));
var _Lock = _interopRequireDefault(require("./Lock"));
var _Trap = _interopRequireDefault(require("./Trap"));
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function _getRequireWildcardCache(e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != _typeof(e) && "function" != typeof e) return { "default": e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && Object.prototype.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n["default"] = e, t && t.set(e, n), n; }
var FocusLockCombination = /*#__PURE__*/(0, _react.forwardRef)(function FocusLockUICombination(props, ref) {
  return /*#__PURE__*/_react["default"].createElement(_Lock["default"], (0, _extends2["default"])({
    sideCar: _Trap["default"],
    ref: ref
  }, props));
});
var _ref = _Lock["default"].propTypes || {},
  sideCar = _ref.sideCar,
  propTypes = (0, _objectWithoutProperties2["default"])(_ref, ["sideCar"]);
FocusLockCombination.propTypes = process.env.NODE_ENV !== "production" ? propTypes : {};
var _default = exports["default"] = FocusLockCombination;