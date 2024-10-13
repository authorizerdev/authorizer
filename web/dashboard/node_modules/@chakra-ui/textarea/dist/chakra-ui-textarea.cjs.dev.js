'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var formControl = require('@chakra-ui/form-control');
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

var _excluded = ["className", "rows"];

/**
 * Textarea is used to enter an amount of text that's longer than a single line
 * @see Docs https://chakra-ui.com/textarea
 */
var Textarea = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyleConfig("Textarea", props);

  var _omitThemingProps = system.omitThemingProps(props),
      className = _omitThemingProps.className,
      rows = _omitThemingProps.rows,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var textareaProps = formControl.useFormControl(rest);
  var omitted = ["h", "minH", "height", "minHeight"];
  var textareaStyles = rows ? utils.omit(styles, omitted) : styles;
  return /*#__PURE__*/React__namespace.createElement(system.chakra.textarea, _extends({
    ref: ref,
    rows: rows
  }, textareaProps, {
    className: utils.cx("chakra-textarea", className),
    __css: textareaStyles
  }));
});

if (utils.__DEV__) {
  Textarea.displayName = "Textarea";
}

exports.Textarea = Textarea;
