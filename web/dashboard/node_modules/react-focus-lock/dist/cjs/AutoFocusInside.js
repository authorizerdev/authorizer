"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports["default"] = void 0;
var _extends2 = _interopRequireDefault(require("@babel/runtime/helpers/extends"));
var _react = _interopRequireDefault(require("react"));
var _propTypes = _interopRequireDefault(require("prop-types"));
var _constants = require("focus-lock/constants");
var _util = require("./util");
var AutoFocusInside = function AutoFocusInside(_ref) {
  var _ref$disabled = _ref.disabled,
    disabled = _ref$disabled === void 0 ? false : _ref$disabled,
    children = _ref.children,
    _ref$className = _ref.className,
    className = _ref$className === void 0 ? undefined : _ref$className;
  return /*#__PURE__*/_react["default"].createElement("div", (0, _extends2["default"])({}, (0, _util.inlineProp)(_constants.FOCUS_AUTO, !disabled), {
    className: className
  }), children);
};
AutoFocusInside.propTypes = process.env.NODE_ENV !== "production" ? {
  children: _propTypes["default"].node.isRequired,
  disabled: _propTypes["default"].bool,
  className: _propTypes["default"].string
} : {};
var _default = exports["default"] = AutoFocusInside;