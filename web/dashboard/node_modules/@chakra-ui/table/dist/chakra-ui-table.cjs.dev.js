'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var React = require('react');

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
var TableContainer = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _ref;

  var overflow = props.overflow,
      overflowX = props.overflowX,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    ref: ref,
    className: utils.cx("chakra-table__container", className)
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
var Table = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("Table", props);

  var _omitThemingProps = system.omitThemingProps(props),
      className = _omitThemingProps.className,
      tableProps = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded2);

  return /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.table, _extends({
    role: "table",
    ref: ref,
    __css: styles.table,
    className: utils.cx("chakra-table", className)
  }, tableProps)));
});

if (utils.__DEV__) {
  Table.displayName = "Table";
}

var TableCaption = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _props$placement = props.placement,
      placement = _props$placement === void 0 ? "bottom" : _props$placement,
      rest = _objectWithoutPropertiesLoose(props, _excluded3);

  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.caption, _extends({}, rest, {
    ref: ref,
    __css: _extends({}, styles.caption, {
      captionSide: placement
    })
  }));
});

if (utils.__DEV__) {
  TableCaption.displayName = "TableCaption";
}

var Thead = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.thead, _extends({}, props, {
    ref: ref,
    __css: styles.thead
  }));
});
var Tbody = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.tbody, _extends({}, props, {
    ref: ref,
    __css: styles.tbody
  }));
});
var Tfoot = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.tfoot, _extends({}, props, {
    ref: ref,
    __css: styles.tfoot
  }));
});
var Th = /*#__PURE__*/system.forwardRef(function (_ref2, ref) {
  var isNumeric = _ref2.isNumeric,
      rest = _objectWithoutPropertiesLoose(_ref2, _excluded4);

  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.th, _extends({}, rest, {
    ref: ref,
    __css: styles.th,
    "data-is-numeric": isNumeric
  }));
});
var Tr = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.tr, _extends({
    role: "row"
  }, props, {
    ref: ref,
    __css: styles.tr
  }));
});
var Td = /*#__PURE__*/system.forwardRef(function (_ref3, ref) {
  var isNumeric = _ref3.isNumeric,
      rest = _objectWithoutPropertiesLoose(_ref3, _excluded5);

  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.td, _extends({
    role: "gridcell"
  }, rest, {
    ref: ref,
    __css: styles.td,
    "data-is-numeric": isNumeric
  }));
});

exports.Table = Table;
exports.TableCaption = TableCaption;
exports.TableContainer = TableContainer;
exports.Tbody = Tbody;
exports.Td = Td;
exports.Tfoot = Tfoot;
exports.Th = Th;
exports.Thead = Thead;
exports.Tr = Tr;
