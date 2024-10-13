'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var icon = require('@chakra-ui/icon');
var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var visuallyHidden = require('@chakra-ui/visually-hidden');
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

var _excluded = ["type", "aria-label"],
    _excluded2 = ["className", "children"];
var StatLabel = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.dt, _extends({
    ref: ref
  }, props, {
    className: utils.cx("chakra-stat__label", props.className),
    __css: styles.label
  }));
});

if (utils.__DEV__) {
  StatLabel.displayName = "StatLabel";
}

var StatHelpText = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.dd, _extends({
    ref: ref
  }, props, {
    className: utils.cx("chakra-stat__help-text", props.className),
    __css: styles.helpText
  }));
});

if (utils.__DEV__) {
  StatHelpText.displayName = "StatHelpText";
}

var StatNumber = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.dd, _extends({
    ref: ref
  }, props, {
    className: utils.cx("chakra-stat__number", props.className),
    __css: _extends({}, styles.number, {
      fontFeatureSettings: "pnum",
      fontVariantNumeric: "proportional-nums"
    })
  }));
});

if (utils.__DEV__) {
  StatNumber.displayName = "StatNumber";
}

var StatDownArrow = function StatDownArrow(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    color: "red.400"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M21,5H3C2.621,5,2.275,5.214,2.105,5.553C1.937,5.892,1.973,6.297,2.2,6.6l9,12 c0.188,0.252,0.485,0.4,0.8,0.4s0.611-0.148,0.8-0.4l9-12c0.228-0.303,0.264-0.708,0.095-1.047C21.725,5.214,21.379,5,21,5z"
  }));
};

if (utils.__DEV__) {
  StatDownArrow.displayName = "StatDownArrow";
}

var StatUpArrow = function StatUpArrow(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    color: "green.400"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M12.8,5.4c-0.377-0.504-1.223-0.504-1.6,0l-9,12c-0.228,0.303-0.264,0.708-0.095,1.047 C2.275,18.786,2.621,19,3,19h18c0.379,0,0.725-0.214,0.895-0.553c0.169-0.339,0.133-0.744-0.095-1.047L12.8,5.4z"
  }));
};

if (utils.__DEV__) {
  StatUpArrow.displayName = "StatUpArrow";
}

var StatArrow = function StatArrow(props) {
  var type = props.type,
      ariaLabel = props["aria-label"],
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var styles = system.useStyles();
  var IconComponent = type === "increase" ? StatUpArrow : StatDownArrow;
  var defaultAriaLabel = type === "increase" ? "increased by" : "decreased by";
  var label = ariaLabel || defaultAriaLabel;
  return /*#__PURE__*/React__namespace.createElement(React__namespace.Fragment, null, /*#__PURE__*/React__namespace.createElement(visuallyHidden.VisuallyHidden, null, label), /*#__PURE__*/React__namespace.createElement(IconComponent, _extends({
    "aria-hidden": true
  }, rest, {
    __css: styles.icon
  })));
};

if (utils.__DEV__) {
  StatArrow.displayName = "StatArrow";
}

var Stat = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("Stat", props);

  var statStyles = _extends({
    position: "relative",
    flex: "1 1 0%"
  }, styles.container);

  var _omitThemingProps = system.omitThemingProps(props),
      className = _omitThemingProps.className,
      children = _omitThemingProps.children,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded2);

  return /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    ref: ref
  }, rest, {
    className: utils.cx("chakra-stat", className),
    __css: statStyles
  }), /*#__PURE__*/React__namespace.createElement("dl", null, children)));
});

if (utils.__DEV__) {
  Stat.displayName = "Stat";
}

var StatGroup = /*#__PURE__*/system.forwardRef(function (props, ref) {
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, props, {
    ref: ref,
    role: "group",
    className: utils.cx("chakra-stat__group", props.className),
    __css: {
      display: "flex",
      flexWrap: "wrap",
      justifyContent: "space-around",
      alignItems: "flex-start"
    }
  }));
});

if (utils.__DEV__) {
  StatGroup.displayName = "StatGroup";
}

exports.Stat = Stat;
exports.StatArrow = StatArrow;
exports.StatDownArrow = StatDownArrow;
exports.StatGroup = StatGroup;
exports.StatHelpText = StatHelpText;
exports.StatLabel = StatLabel;
exports.StatNumber = StatNumber;
exports.StatUpArrow = StatUpArrow;
