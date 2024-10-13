'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

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

var _excluded = ["label", "thickness", "speed", "emptyColor", "className"];
var spin = system.keyframes({
  "0%": {
    transform: "rotate(0deg)"
  },
  "100%": {
    transform: "rotate(360deg)"
  }
});

/**
 * Spinner is used to indicate the loading state of a page or a component,
 * It renders a `div` by default.
 *
 * @see Docs https://chakra-ui.com/spinner
 */
var Spinner = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyleConfig("Spinner", props);

  var _omitThemingProps = system.omitThemingProps(props),
      _omitThemingProps$lab = _omitThemingProps.label,
      label = _omitThemingProps$lab === void 0 ? "Loading..." : _omitThemingProps$lab,
      _omitThemingProps$thi = _omitThemingProps.thickness,
      thickness = _omitThemingProps$thi === void 0 ? "2px" : _omitThemingProps$thi,
      _omitThemingProps$spe = _omitThemingProps.speed,
      speed = _omitThemingProps$spe === void 0 ? "0.45s" : _omitThemingProps$spe,
      _omitThemingProps$emp = _omitThemingProps.emptyColor,
      emptyColor = _omitThemingProps$emp === void 0 ? "transparent" : _omitThemingProps$emp,
      className = _omitThemingProps.className,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var _className = utils.cx("chakra-spinner", className);

  var spinnerStyles = _extends({
    display: "inline-block",
    borderColor: "currentColor",
    borderStyle: "solid",
    borderRadius: "99999px",
    borderWidth: thickness,
    borderBottomColor: emptyColor,
    borderLeftColor: emptyColor,
    animation: spin + " " + speed + " linear infinite"
  }, styles);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    ref: ref,
    __css: spinnerStyles,
    className: _className
  }, rest), label && /*#__PURE__*/React__namespace.createElement(visuallyHidden.VisuallyHidden, null, label));
});

if (utils.__DEV__) {
  Spinner.displayName = "Spinner";
}

exports.Spinner = Spinner;
