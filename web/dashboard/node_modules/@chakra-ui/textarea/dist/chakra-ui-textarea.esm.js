import { useFormControl } from '@chakra-ui/form-control';
import { forwardRef, useStyleConfig, omitThemingProps, chakra } from '@chakra-ui/system';
import { omit, cx, __DEV__ } from '@chakra-ui/utils';
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

var _excluded = ["className", "rows"];

/**
 * Textarea is used to enter an amount of text that's longer than a single line
 * @see Docs https://chakra-ui.com/textarea
 */
var Textarea = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Textarea", props);

  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      rows = _omitThemingProps.rows,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var textareaProps = useFormControl(rest);
  var omitted = ["h", "minH", "height", "minHeight"];
  var textareaStyles = rows ? omit(styles, omitted) : styles;
  return /*#__PURE__*/React.createElement(chakra.textarea, _extends({
    ref: ref,
    rows: rows
  }, textareaProps, {
    className: cx("chakra-textarea", className),
    __css: textareaStyles
  }));
});

if (__DEV__) {
  Textarea.displayName = "Textarea";
}

export { Textarea };
