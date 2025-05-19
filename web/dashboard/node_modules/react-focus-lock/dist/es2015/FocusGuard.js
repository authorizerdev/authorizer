import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
export var hiddenGuard = {
  width: '1px',
  height: '0px',
  padding: 0,
  overflow: 'hidden',
  position: 'fixed',
  top: '1px',
  left: '1px'
};
var InFocusGuard = function InFocusGuard(_ref) {
  var _ref$children = _ref.children,
    children = _ref$children === void 0 ? null : _ref$children;
  return /*#__PURE__*/React.createElement(Fragment, null, /*#__PURE__*/React.createElement("div", {
    key: "guard-first",
    "data-focus-guard": true,
    "data-focus-auto-guard": true,
    style: hiddenGuard
  }), children, children && /*#__PURE__*/React.createElement("div", {
    key: "guard-last",
    "data-focus-guard": true,
    "data-focus-auto-guard": true,
    style: hiddenGuard
  }));
};
InFocusGuard.propTypes = process.env.NODE_ENV !== "production" ? {
  children: PropTypes.node
} : {};
export default InFocusGuard;