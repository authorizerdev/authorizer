import { useMergeRefs } from '@chakra-ui/hooks';
import { forwardRef, chakra, useStyleConfig, omitThemingProps } from '@chakra-ui/system';
import { cx, __DEV__, mergeWith, dataAttr } from '@chakra-ui/utils';
import * as React from 'react';
import { createContext } from '@chakra-ui/react-utils';
import { Spinner } from '@chakra-ui/spinner';

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

var _excluded$4 = ["size", "colorScheme", "variant", "className", "spacing", "isAttached", "isDisabled"];

var _createContext = createContext({
  strict: false,
  name: "ButtonGroupContext"
}),
    ButtonGroupProvider = _createContext[0],
    useButtonGroup = _createContext[1];
var ButtonGroup = /*#__PURE__*/forwardRef(function (props, ref) {
  var size = props.size,
      colorScheme = props.colorScheme,
      variant = props.variant,
      className = props.className,
      _props$spacing = props.spacing,
      spacing = _props$spacing === void 0 ? "0.5rem" : _props$spacing,
      isAttached = props.isAttached,
      isDisabled = props.isDisabled,
      rest = _objectWithoutPropertiesLoose(props, _excluded$4);

  var _className = cx("chakra-button__group", className);

  var context = React.useMemo(function () {
    return {
      size: size,
      colorScheme: colorScheme,
      variant: variant,
      isDisabled: isDisabled
    };
  }, [size, colorScheme, variant, isDisabled]);
  var groupStyles = {
    display: "inline-flex"
  };

  if (isAttached) {
    groupStyles = _extends({}, groupStyles, {
      "> *:first-of-type:not(:last-of-type)": {
        borderEndRadius: 0
      },
      "> *:not(:first-of-type):not(:last-of-type)": {
        borderRadius: 0
      },
      "> *:not(:first-of-type):last-of-type": {
        borderStartRadius: 0
      }
    });
  } else {
    groupStyles = _extends({}, groupStyles, {
      "& > *:not(style) ~ *:not(style)": {
        marginStart: spacing
      }
    });
  }

  return /*#__PURE__*/React.createElement(ButtonGroupProvider, {
    value: context
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    role: "group",
    __css: groupStyles,
    className: _className
  }, rest)));
});

if (__DEV__) {
  ButtonGroup.displayName = "ButtonGroup";
}

var _excluded$3 = ["label", "placement", "spacing", "children", "className", "__css"];
var ButtonSpinner = function ButtonSpinner(props) {
  var label = props.label,
      placement = props.placement;
      props.spacing;
      var _props$children = props.children,
      children = _props$children === void 0 ? /*#__PURE__*/React.createElement(Spinner, {
    color: "currentColor",
    width: "1em",
    height: "1em"
  }) : _props$children,
      className = props.className,
      __css = props.__css,
      rest = _objectWithoutPropertiesLoose(props, _excluded$3);

  var _className = cx("chakra-button__spinner", className);

  var marginProp = placement === "start" ? "marginEnd" : "marginStart";
  var spinnerStyles = React.useMemo(function () {
    var _extends2;

    return _extends((_extends2 = {
      display: "flex",
      alignItems: "center",
      position: label ? "relative" : "absolute"
    }, _extends2[marginProp] = label ? "0.5rem" : 0, _extends2.fontSize = "1em", _extends2.lineHeight = "normal", _extends2), __css);
  }, [__css, label, marginProp]);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    className: _className
  }, rest, {
    __css: spinnerStyles
  }), children);
};

if (__DEV__) {
  ButtonSpinner.displayName = "ButtonSpinner";
}

var _excluded$2 = ["children", "className"];
var ButtonIcon = function ButtonIcon(props) {
  var children = props.children,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded$2);

  var _children = /*#__PURE__*/React.isValidElement(children) ? /*#__PURE__*/React.cloneElement(children, {
    "aria-hidden": true,
    focusable: false
  }) : children;

  var _className = cx("chakra-button__icon", className);

  return /*#__PURE__*/React.createElement(chakra.span, _extends({
    display: "inline-flex",
    alignSelf: "center",
    flexShrink: 0
  }, rest, {
    className: _className
  }), _children);
};

if (__DEV__) {
  ButtonIcon.displayName = "ButtonIcon";
}

function useButtonType(value) {
  var _React$useState = React.useState(!value),
      isButton = _React$useState[0],
      setIsButton = _React$useState[1];

  var refCallback = React.useCallback(function (node) {
    if (!node) return;
    setIsButton(node.tagName === "BUTTON");
  }, []);
  var type = isButton ? "button" : undefined;
  return {
    ref: refCallback,
    type: type
  };
}

var _excluded$1 = ["isDisabled", "isLoading", "isActive", "isFullWidth", "children", "leftIcon", "rightIcon", "loadingText", "iconSpacing", "type", "spinner", "spinnerPlacement", "className", "as"];
var Button = /*#__PURE__*/forwardRef(function (props, ref) {
  var group = useButtonGroup();
  var styles = useStyleConfig("Button", _extends({}, group, props));

  var _omitThemingProps = omitThemingProps(props),
      _omitThemingProps$isD = _omitThemingProps.isDisabled,
      isDisabled = _omitThemingProps$isD === void 0 ? group == null ? void 0 : group.isDisabled : _omitThemingProps$isD,
      isLoading = _omitThemingProps.isLoading,
      isActive = _omitThemingProps.isActive,
      isFullWidth = _omitThemingProps.isFullWidth,
      children = _omitThemingProps.children,
      leftIcon = _omitThemingProps.leftIcon,
      rightIcon = _omitThemingProps.rightIcon,
      loadingText = _omitThemingProps.loadingText,
      _omitThemingProps$ico = _omitThemingProps.iconSpacing,
      iconSpacing = _omitThemingProps$ico === void 0 ? "0.5rem" : _omitThemingProps$ico,
      type = _omitThemingProps.type,
      spinner = _omitThemingProps.spinner,
      _omitThemingProps$spi = _omitThemingProps.spinnerPlacement,
      spinnerPlacement = _omitThemingProps$spi === void 0 ? "start" : _omitThemingProps$spi,
      className = _omitThemingProps.className,
      as = _omitThemingProps.as,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$1);
  /**
   * When button is used within ButtonGroup (i.e flushed with sibling buttons),
   * it is important to add a `zIndex` on focus.
   *
   * So let's read the component styles and then add `zIndex` to it.
   */


  var buttonStyles = React.useMemo(function () {
    var _styles$_focus;

    var _focus = mergeWith({}, (_styles$_focus = styles == null ? void 0 : styles["_focus"]) != null ? _styles$_focus : {}, {
      zIndex: 1
    });

    return _extends({
      display: "inline-flex",
      appearance: "none",
      alignItems: "center",
      justifyContent: "center",
      userSelect: "none",
      position: "relative",
      whiteSpace: "nowrap",
      verticalAlign: "middle",
      outline: "none",
      width: isFullWidth ? "100%" : "auto"
    }, styles, !!group && {
      _focus: _focus
    });
  }, [styles, group, isFullWidth]);

  var _useButtonType = useButtonType(as),
      _ref = _useButtonType.ref,
      defaultType = _useButtonType.type;

  var contentProps = {
    rightIcon: rightIcon,
    leftIcon: leftIcon,
    iconSpacing: iconSpacing,
    children: children
  };
  return /*#__PURE__*/React.createElement(chakra.button, _extends({
    disabled: isDisabled || isLoading,
    ref: useMergeRefs(ref, _ref),
    as: as,
    type: type != null ? type : defaultType,
    "data-active": dataAttr(isActive),
    "data-loading": dataAttr(isLoading),
    __css: buttonStyles,
    className: cx("chakra-button", className)
  }, rest), isLoading && spinnerPlacement === "start" && /*#__PURE__*/React.createElement(ButtonSpinner, {
    className: "chakra-button__spinner--start",
    label: loadingText,
    placement: "start"
  }, spinner), isLoading ? loadingText || /*#__PURE__*/React.createElement(chakra.span, {
    opacity: 0
  }, /*#__PURE__*/React.createElement(ButtonContent, contentProps)) : /*#__PURE__*/React.createElement(ButtonContent, contentProps), isLoading && spinnerPlacement === "end" && /*#__PURE__*/React.createElement(ButtonSpinner, {
    className: "chakra-button__spinner--end",
    label: loadingText,
    placement: "end"
  }, spinner));
});

if (__DEV__) {
  Button.displayName = "Button";
}

function ButtonContent(props) {
  var leftIcon = props.leftIcon,
      rightIcon = props.rightIcon,
      children = props.children,
      iconSpacing = props.iconSpacing;
  return /*#__PURE__*/React.createElement(React.Fragment, null, leftIcon && /*#__PURE__*/React.createElement(ButtonIcon, {
    marginEnd: iconSpacing
  }, leftIcon), children, rightIcon && /*#__PURE__*/React.createElement(ButtonIcon, {
    marginStart: iconSpacing
  }, rightIcon));
}

var _excluded = ["icon", "children", "isRound", "aria-label"];
var IconButton = /*#__PURE__*/forwardRef(function (props, ref) {
  var icon = props.icon,
      children = props.children,
      isRound = props.isRound,
      ariaLabel = props["aria-label"],
      rest = _objectWithoutPropertiesLoose(props, _excluded);
  /**
   * Passing the icon as prop or children should work
   */


  var element = icon || children;

  var _children = /*#__PURE__*/React.isValidElement(element) ? /*#__PURE__*/React.cloneElement(element, {
    "aria-hidden": true,
    focusable: false
  }) : null;

  return /*#__PURE__*/React.createElement(Button, _extends({
    padding: "0",
    borderRadius: isRound ? "full" : undefined,
    ref: ref,
    "aria-label": ariaLabel
  }, rest), _children);
});

if (__DEV__) {
  IconButton.displayName = "IconButton";
}

export { Button, ButtonGroup, ButtonSpinner, IconButton, useButtonGroup };
