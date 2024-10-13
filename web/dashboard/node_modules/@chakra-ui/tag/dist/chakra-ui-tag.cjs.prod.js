'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var icon = require('@chakra-ui/icon');
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

var _excluded = ["isDisabled", "children"];

/**
 * The tag component is used to label or categorize UI elements.
 * To style the tag globally, change the styles in `theme.components.Tag`
 * @see Docs https://chakra-ui.com/tag
 */
var Tag = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("Tag", props);
  var ownProps = system.omitThemingProps(props);

  var containerStyles = _extends({
    display: "inline-flex",
    verticalAlign: "top",
    alignItems: "center",
    maxWidth: "100%"
  }, styles.container);

  return /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    ref: ref
  }, ownProps, {
    __css: containerStyles
  })));
});

if (utils.__DEV__) {
  Tag.displayName = "Tag";
}

var TagLabel = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    ref: ref,
    isTruncated: true
  }, props, {
    __css: styles.label
  }));
});

if (utils.__DEV__) {
  TagLabel.displayName = "TagLabel";
}

var TagLeftIcon = /*#__PURE__*/system.forwardRef(function (props, ref) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    ref: ref,
    verticalAlign: "top",
    marginEnd: "0.5rem"
  }, props));
});

if (utils.__DEV__) {
  TagLeftIcon.displayName = "TagLeftIcon";
}

var TagRightIcon = /*#__PURE__*/system.forwardRef(function (props, ref) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    ref: ref,
    verticalAlign: "top",
    marginStart: "0.5rem"
  }, props));
});

if (utils.__DEV__) {
  TagRightIcon.displayName = "TagRightIcon";
}

var TagCloseIcon = function TagCloseIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    verticalAlign: "inherit",
    viewBox: "0 0 512 512"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M289.94 256l95-95A24 24 0 00351 127l-95 95-95-95a24 24 0 00-34 34l95 95-95 95a24 24 0 1034 34l95-95 95 95a24 24 0 0034-34z"
  }));
};

if (utils.__DEV__) {
  TagCloseIcon.displayName = "TagCloseIcon";
}

/**
 * TagCloseButton is used to close "remove" the tag
 * @see Docs https://chakra-ui.com/tag
 */
var TagCloseButton = function TagCloseButton(props) {
  var isDisabled = props.isDisabled,
      children = props.children,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var styles = system.useStyles();

  var btnStyles = _extends({
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    outline: "0"
  }, styles.closeButton);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.button, _extends({}, rest, {
    type: "button",
    "aria-label": "close",
    disabled: isDisabled,
    __css: btnStyles
  }), children || /*#__PURE__*/React__namespace.createElement(TagCloseIcon, null));
};

if (utils.__DEV__) {
  TagCloseButton.displayName = "TagCloseButton";
}

exports.Tag = Tag;
exports.TagCloseButton = TagCloseButton;
exports.TagLabel = TagLabel;
exports.TagLeftIcon = TagLeftIcon;
exports.TagRightIcon = TagRightIcon;
