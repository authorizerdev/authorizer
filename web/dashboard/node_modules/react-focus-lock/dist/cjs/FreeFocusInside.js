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
var FreeFocusInside = function FreeFocusInside(_ref) {
  var children = _ref.children,
    className = _ref.className;
  return /*#__PURE__*/_react["default"].createElement("div", (0, _extends2["default"])({}, (0, _util.inlineProp)(_constants.FOCUS_ALLOW, true), {
    className: className
  }), children);
};
FreeFocusInside.propTypes = process.env.NODE_ENV !== "production" ? {
  children: _propTypes["default"].node.isRequired,
  className: _propTypes["default"].string
} : {};
var _default = exports["default"] = FreeFocusInside;