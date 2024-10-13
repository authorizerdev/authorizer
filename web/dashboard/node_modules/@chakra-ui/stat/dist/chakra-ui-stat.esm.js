import { Icon } from '@chakra-ui/icon';
import { forwardRef, useStyles, chakra, useMultiStyleConfig, omitThemingProps, StylesProvider } from '@chakra-ui/system';
import { cx, __DEV__ } from '@chakra-ui/utils';
import { VisuallyHidden } from '@chakra-ui/visually-hidden';
import * as React from 'react';

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
var StatLabel = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.dt, _extends({
    ref: ref
  }, props, {
    className: cx("chakra-stat__label", props.className),
    __css: styles.label
  }));
});

if (__DEV__) {
  StatLabel.displayName = "StatLabel";
}

var StatHelpText = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.dd, _extends({
    ref: ref
  }, props, {
    className: cx("chakra-stat__help-text", props.className),
    __css: styles.helpText
  }));
});

if (__DEV__) {
  StatHelpText.displayName = "StatHelpText";
}

var StatNumber = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.dd, _extends({
    ref: ref
  }, props, {
    className: cx("chakra-stat__number", props.className),
    __css: _extends({}, styles.number, {
      fontFeatureSettings: "pnum",
      fontVariantNumeric: "proportional-nums"
    })
  }));
});

if (__DEV__) {
  StatNumber.displayName = "StatNumber";
}

var StatDownArrow = function StatDownArrow(props) {
  return /*#__PURE__*/React.createElement(Icon, _extends({
    color: "red.400"
  }, props), /*#__PURE__*/React.createElement("path", {
    fill: "currentColor",
    d: "M21,5H3C2.621,5,2.275,5.214,2.105,5.553C1.937,5.892,1.973,6.297,2.2,6.6l9,12 c0.188,0.252,0.485,0.4,0.8,0.4s0.611-0.148,0.8-0.4l9-12c0.228-0.303,0.264-0.708,0.095-1.047C21.725,5.214,21.379,5,21,5z"
  }));
};

if (__DEV__) {
  StatDownArrow.displayName = "StatDownArrow";
}

var StatUpArrow = function StatUpArrow(props) {
  return /*#__PURE__*/React.createElement(Icon, _extends({
    color: "green.400"
  }, props), /*#__PURE__*/React.createElement("path", {
    fill: "currentColor",
    d: "M12.8,5.4c-0.377-0.504-1.223-0.504-1.6,0l-9,12c-0.228,0.303-0.264,0.708-0.095,1.047 C2.275,18.786,2.621,19,3,19h18c0.379,0,0.725-0.214,0.895-0.553c0.169-0.339,0.133-0.744-0.095-1.047L12.8,5.4z"
  }));
};

if (__DEV__) {
  StatUpArrow.displayName = "StatUpArrow";
}

var StatArrow = function StatArrow(props) {
  var type = props.type,
      ariaLabel = props["aria-label"],
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var styles = useStyles();
  var IconComponent = type === "increase" ? StatUpArrow : StatDownArrow;
  var defaultAriaLabel = type === "increase" ? "increased by" : "decreased by";
  var label = ariaLabel || defaultAriaLabel;
  return /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement(VisuallyHidden, null, label), /*#__PURE__*/React.createElement(IconComponent, _extends({
    "aria-hidden": true
  }, rest, {
    __css: styles.icon
  })));
};

if (__DEV__) {
  StatArrow.displayName = "StatArrow";
}

var Stat = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Stat", props);

  var statStyles = _extends({
    position: "relative",
    flex: "1 1 0%"
  }, styles.container);

  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      children = _omitThemingProps.children,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded2);

  return /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref
  }, rest, {
    className: cx("chakra-stat", className),
    __css: statStyles
  }), /*#__PURE__*/React.createElement("dl", null, children)));
});

if (__DEV__) {
  Stat.displayName = "Stat";
}

var StatGroup = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, props, {
    ref: ref,
    role: "group",
    className: cx("chakra-stat__group", props.className),
    __css: {
      display: "flex",
      flexWrap: "wrap",
      justifyContent: "space-around",
      alignItems: "flex-start"
    }
  }));
});

if (__DEV__) {
  StatGroup.displayName = "StatGroup";
}

export { Stat, StatArrow, StatDownArrow, StatGroup, StatHelpText, StatLabel, StatNumber, StatUpArrow };
