import { forwardRef, chakra, useMultiStyleConfig, omitThemingProps, StylesProvider, useStyles } from '@chakra-ui/system';
import { cx, __DEV__ } from '@chakra-ui/utils';
import * as React from 'react';

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

var _excluded = ["overflow", "overflowX", "className"],
    _excluded2 = ["className"],
    _excluded3 = ["placement"],
    _excluded4 = ["isNumeric"],
    _excluded5 = ["isNumeric"];
var TableContainer = /*#__PURE__*/forwardRef(function (props, ref) {
  var _ref;

  var overflow = props.overflow,
      overflowX = props.overflowX,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    className: cx("chakra-table__container", className)
  }, rest, {
    __css: {
      display: "block",
      whiteSpace: "nowrap",
      WebkitOverflowScrolling: "touch",
      overflowX: (_ref = overflow != null ? overflow : overflowX) != null ? _ref : "auto",
      overflowY: "hidden",
      maxWidth: "100%"
    }
  }));
});
var Table = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Table", props);

  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      tableProps = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded2);

  return /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.table, _extends({
    role: "table",
    ref: ref,
    __css: styles.table,
    className: cx("chakra-table", className)
  }, tableProps)));
});

if (__DEV__) {
  Table.displayName = "Table";
}

var TableCaption = /*#__PURE__*/forwardRef(function (props, ref) {
  var _props$placement = props.placement,
      placement = _props$placement === void 0 ? "bottom" : _props$placement,
      rest = _objectWithoutPropertiesLoose(props, _excluded3);

  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.caption, _extends({}, rest, {
    ref: ref,
    __css: _extends({}, styles.caption, {
      captionSide: placement
    })
  }));
});

if (__DEV__) {
  TableCaption.displayName = "TableCaption";
}

var Thead = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.thead, _extends({}, props, {
    ref: ref,
    __css: styles.thead
  }));
});
var Tbody = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.tbody, _extends({}, props, {
    ref: ref,
    __css: styles.tbody
  }));
});
var Tfoot = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.tfoot, _extends({}, props, {
    ref: ref,
    __css: styles.tfoot
  }));
});
var Th = /*#__PURE__*/forwardRef(function (_ref2, ref) {
  var isNumeric = _ref2.isNumeric,
      rest = _objectWithoutPropertiesLoose(_ref2, _excluded4);

  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.th, _extends({}, rest, {
    ref: ref,
    __css: styles.th,
    "data-is-numeric": isNumeric
  }));
});
var Tr = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.tr, _extends({
    role: "row"
  }, props, {
    ref: ref,
    __css: styles.tr
  }));
});
var Td = /*#__PURE__*/forwardRef(function (_ref3, ref) {
  var isNumeric = _ref3.isNumeric,
      rest = _objectWithoutPropertiesLoose(_ref3, _excluded5);

  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.td, _extends({
    role: "gridcell"
  }, rest, {
    ref: ref,
    __css: styles.td,
    "data-is-numeric": isNumeric
  }));
});

export { Table, TableCaption, TableContainer, Tbody, Td, Tfoot, Th, Thead, Tr };
