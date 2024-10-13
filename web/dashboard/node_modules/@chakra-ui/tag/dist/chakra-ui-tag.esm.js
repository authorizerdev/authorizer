import { Icon } from '@chakra-ui/icon';
import { forwardRef, useMultiStyleConfig, omitThemingProps, StylesProvider, chakra, useStyles } from '@chakra-ui/system';
import { __DEV__ } from '@chakra-ui/utils';
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

var _excluded = ["isDisabled", "children"];

/**
 * The tag component is used to label or categorize UI elements.
 * To style the tag globally, change the styles in `theme.components.Tag`
 * @see Docs https://chakra-ui.com/tag
 */
var Tag = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Tag", props);
  var ownProps = omitThemingProps(props);

  var containerStyles = _extends({
    display: "inline-flex",
    verticalAlign: "top",
    alignItems: "center",
    maxWidth: "100%"
  }, styles.container);

  return /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.span, _extends({
    ref: ref
  }, ownProps, {
    __css: containerStyles
  })));
});

if (__DEV__) {
  Tag.displayName = "Tag";
}

var TagLabel = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.span, _extends({
    ref: ref,
    isTruncated: true
  }, props, {
    __css: styles.label
  }));
});

if (__DEV__) {
  TagLabel.displayName = "TagLabel";
}

var TagLeftIcon = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(Icon, _extends({
    ref: ref,
    verticalAlign: "top",
    marginEnd: "0.5rem"
  }, props));
});

if (__DEV__) {
  TagLeftIcon.displayName = "TagLeftIcon";
}

var TagRightIcon = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(Icon, _extends({
    ref: ref,
    verticalAlign: "top",
    marginStart: "0.5rem"
  }, props));
});

if (__DEV__) {
  TagRightIcon.displayName = "TagRightIcon";
}

var TagCloseIcon = function TagCloseIcon(props) {
  return /*#__PURE__*/React.createElement(Icon, _extends({
    verticalAlign: "inherit",
    viewBox: "0 0 512 512"
  }, props), /*#__PURE__*/React.createElement("path", {
    fill: "currentColor",
    d: "M289.94 256l95-95A24 24 0 00351 127l-95 95-95-95a24 24 0 00-34 34l95 95-95 95a24 24 0 1034 34l95-95 95 95a24 24 0 0034-34z"
  }));
};

if (__DEV__) {
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

  var styles = useStyles();

  var btnStyles = _extends({
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    outline: "0"
  }, styles.closeButton);

  return /*#__PURE__*/React.createElement(chakra.button, _extends({}, rest, {
    type: "button",
    "aria-label": "close",
    disabled: isDisabled,
    __css: btnStyles
  }), children || /*#__PURE__*/React.createElement(TagCloseIcon, null));
};

if (__DEV__) {
  TagCloseButton.displayName = "TagCloseButton";
}

export { Tag, TagCloseButton, TagLabel, TagLeftIcon, TagRightIcon };
