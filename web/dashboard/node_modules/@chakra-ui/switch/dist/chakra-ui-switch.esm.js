import { useCheckbox } from '@chakra-ui/checkbox';
import { forwardRef, useMultiStyleConfig, omitThemingProps, chakra } from '@chakra-ui/system';
import { cx, dataAttr, __DEV__ } from '@chakra-ui/utils';
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

var _excluded = ["spacing", "children"];
var Switch = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Switch", props);

  var _omitThemingProps = omitThemingProps(props),
      _omitThemingProps$spa = _omitThemingProps.spacing,
      spacing = _omitThemingProps$spa === void 0 ? "0.5rem" : _omitThemingProps$spa,
      children = _omitThemingProps.children,
      ownProps = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var _useCheckbox = useCheckbox(ownProps),
      state = _useCheckbox.state,
      getInputProps = _useCheckbox.getInputProps,
      getCheckboxProps = _useCheckbox.getCheckboxProps,
      getRootProps = _useCheckbox.getRootProps,
      getLabelProps = _useCheckbox.getLabelProps;

  var containerStyles = React.useMemo(function () {
    return _extends({
      display: "inline-block",
      position: "relative",
      verticalAlign: "middle",
      lineHeight: "normal"
    }, styles.container);
  }, [styles.container]);
  var trackStyles = React.useMemo(function () {
    return _extends({
      display: "inline-flex",
      flexShrink: 0,
      justifyContent: "flex-start",
      boxSizing: "content-box",
      cursor: "pointer"
    }, styles.track);
  }, [styles.track]);
  var labelStyles = React.useMemo(function () {
    return _extends({
      userSelect: "none",
      marginStart: spacing
    }, styles.label);
  }, [spacing, styles.label]);
  return /*#__PURE__*/React.createElement(chakra.label, _extends({}, getRootProps(), {
    className: cx("chakra-switch", props.className),
    __css: containerStyles
  }), /*#__PURE__*/React.createElement("input", _extends({
    className: "chakra-switch__input"
  }, getInputProps({}, ref))), /*#__PURE__*/React.createElement(chakra.span, _extends({}, getCheckboxProps(), {
    className: "chakra-switch__track",
    __css: trackStyles
  }), /*#__PURE__*/React.createElement(chakra.span, {
    __css: styles.thumb,
    className: "chakra-switch__thumb",
    "data-checked": dataAttr(state.isChecked),
    "data-hover": dataAttr(state.isHovered)
  })), children && /*#__PURE__*/React.createElement(chakra.span, _extends({
    className: "chakra-switch__label"
  }, getLabelProps(), {
    __css: labelStyles
  }), children));
});

if (__DEV__) {
  Switch.displayName = "Switch";
}

export { Switch };
