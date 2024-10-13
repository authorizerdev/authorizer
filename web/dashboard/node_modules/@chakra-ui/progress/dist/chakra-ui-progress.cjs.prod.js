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

var spin = system.keyframes({
  "0%": {
    strokeDasharray: "1, 400",
    strokeDashoffset: "0"
  },
  "50%": {
    strokeDasharray: "400, 400",
    strokeDashoffset: "-100"
  },
  "100%": {
    strokeDasharray: "400, 400",
    strokeDashoffset: "-260"
  }
});
var rotate = system.keyframes({
  "0%": {
    transform: "rotate(0deg)"
  },
  "100%": {
    transform: "rotate(360deg)"
  }
});
var progress = system.keyframes({
  "0%": {
    left: "-40%"
  },
  "100%": {
    left: "100%"
  }
});
var stripe = system.keyframes({
  from: {
    backgroundPosition: "1rem 0"
  },
  to: {
    backgroundPosition: "0 0"
  }
});

/**
 * Get the common `aria-*` attributes for both the linear and circular
 * progress components.
 */
function getProgressProps(options) {
  var _options$value = options.value,
      value = _options$value === void 0 ? 0 : _options$value,
      min = options.min,
      max = options.max,
      valueText = options.valueText,
      getValueText = options.getValueText,
      isIndeterminate = options.isIndeterminate;
  var percent = utils.valueToPercent(value, min, max);

  var getAriaValueText = function getAriaValueText() {
    if (value == null) return undefined;
    return utils.isFunction(getValueText) ? getValueText(value, percent) : valueText;
  };

  return {
    bind: {
      "data-indeterminate": isIndeterminate ? "" : undefined,
      "aria-valuemax": max,
      "aria-valuemin": min,
      "aria-valuenow": isIndeterminate ? undefined : value,
      "aria-valuetext": getAriaValueText(),
      role: "progressbar"
    },
    percent: percent,
    value: value
  };
}

var _excluded$1 = ["size", "isIndeterminate"],
    _excluded2$1 = ["size", "max", "min", "valueText", "getValueText", "value", "capIsRound", "children", "thickness", "color", "trackColor", "isIndeterminate"];

var Circle = function Circle(props) {
  return /*#__PURE__*/React__namespace.createElement(system.chakra.circle, _extends({
    cx: 50,
    cy: 50,
    r: 42,
    fill: "transparent"
  }, props));
};

if (utils.__DEV__) {
  Circle.displayName = "Circle";
}

var Shape = function Shape(props) {
  var size = props.size,
      isIndeterminate = props.isIndeterminate,
      rest = _objectWithoutPropertiesLoose(props, _excluded$1);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.svg, _extends({
    viewBox: "0 0 100 100",
    __css: {
      width: size,
      height: size,
      animation: isIndeterminate ? rotate + " 2s linear infinite" : undefined
    }
  }, rest));
};

if (utils.__DEV__) {
  Shape.displayName = "Shape";
}

/**
 * CircularProgress is used to indicate the progress of an activity.
 * It is built using `svg` and `circle` components with support for
 * theming and `indeterminate` state
 *
 * @see Docs https://chakra-ui.com/circularprogress
 * @todo add theming support for circular progress
 */
var CircularProgress = function CircularProgress(props) {
  var _progress$percent;

  var _props$size = props.size,
      size = _props$size === void 0 ? "48px" : _props$size,
      _props$max = props.max,
      max = _props$max === void 0 ? 100 : _props$max,
      _props$min = props.min,
      min = _props$min === void 0 ? 0 : _props$min,
      valueText = props.valueText,
      getValueText = props.getValueText,
      value = props.value,
      capIsRound = props.capIsRound,
      children = props.children,
      _props$thickness = props.thickness,
      thickness = _props$thickness === void 0 ? "10px" : _props$thickness,
      _props$color = props.color,
      color = _props$color === void 0 ? "#0078d4" : _props$color,
      _props$trackColor = props.trackColor,
      trackColor = _props$trackColor === void 0 ? "#edebe9" : _props$trackColor,
      isIndeterminate = props.isIndeterminate,
      rest = _objectWithoutPropertiesLoose(props, _excluded2$1);

  var progress = getProgressProps({
    min: min,
    max: max,
    value: value,
    valueText: valueText,
    getValueText: getValueText,
    isIndeterminate: isIndeterminate
  });
  var determinant = isIndeterminate ? undefined : ((_progress$percent = progress.percent) != null ? _progress$percent : 0) * 2.64;
  var strokeDasharray = utils.isUndefined(determinant) ? undefined : determinant + " " + (264 - determinant);
  var indicatorProps = isIndeterminate ? {
    css: {
      animation: spin + " 1.5s linear infinite"
    }
  } : {
    strokeDashoffset: 66,
    strokeDasharray: strokeDasharray,
    transitionProperty: "stroke-dasharray, stroke",
    transitionDuration: "0.6s",
    transitionTimingFunction: "ease"
  };
  var rootStyles = {
    display: "inline-block",
    position: "relative",
    verticalAlign: "middle",
    fontSize: size
  };
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    className: "chakra-progress"
  }, progress.bind, rest, {
    __css: rootStyles
  }), /*#__PURE__*/React__namespace.createElement(Shape, {
    size: size,
    isIndeterminate: isIndeterminate
  }, /*#__PURE__*/React__namespace.createElement(Circle, {
    stroke: trackColor,
    strokeWidth: thickness,
    className: "chakra-progress__track"
  }), /*#__PURE__*/React__namespace.createElement(Circle, _extends({
    stroke: color,
    strokeWidth: thickness,
    className: "chakra-progress__indicator",
    strokeLinecap: capIsRound ? "round" : undefined
    /**
     * fix issue in Safari where indictor still shows when value is 0
     * @see Issue https://github.com/chakra-ui/chakra-ui/issues/3754
     */
    ,
    opacity: progress.value === 0 && !isIndeterminate ? 0 : undefined
  }, indicatorProps))), children);
};

if (utils.__DEV__) {
  CircularProgress.displayName = "CircularProgress";
}
/**
 * CircularProgress component label. In most cases it is a numeric indicator
 * of the circular progress component's value
 */


var CircularProgressLabel = system.chakra("div", {
  baseStyle: {
    fontSize: "0.24em",
    top: "50%",
    left: "50%",
    width: "100%",
    textAlign: "center",
    position: "absolute",
    transform: "translate(-50%, -50%)"
  }
});

if (utils.__DEV__) {
  CircularProgressLabel.displayName = "CircularProgressLabel";
}

var _excluded = ["min", "max", "value", "isIndeterminate"],
    _excluded2 = ["value", "min", "max", "hasStripe", "isAnimated", "children", "borderRadius", "isIndeterminate", "aria-label", "aria-labelledby"];

/**
 * ProgressLabel is used to show the numeric value of the progress.
 * @see Docs https://chakra-ui.com/progress
 */
var ProgressLabel = function ProgressLabel(props) {
  var styles = system.useStyles();

  var labelStyles = _extends({
    top: "50%",
    left: "50%",
    width: "100%",
    textAlign: "center",
    position: "absolute",
    transform: "translate(-50%, -50%)"
  }, styles.label);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, props, {
    __css: labelStyles
  }));
};

if (utils.__DEV__) {
  ProgressLabel.displayName = "ProgressLabel";
}

/**
 * ProgressFilledTrack (Linear)
 *
 * The progress component that visually indicates the current level of the progress bar.
 * It applies `background-color` and changes its width.
 *
 * @see Docs https://chakra-ui.com/progress
 */
var ProgressFilledTrack = function ProgressFilledTrack(props) {
  var min = props.min,
      max = props.max,
      value = props.value,
      isIndeterminate = props.isIndeterminate,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var progress = getProgressProps({
    value: value,
    min: min,
    max: max,
    isIndeterminate: isIndeterminate
  });
  var styles = system.useStyles();

  var trackStyles = _extends({
    height: "100%"
  }, styles.filledTrack);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    style: _extends({
      width: progress.percent + "%"
    }, rest.style)
  }, progress.bind, rest, {
    __css: trackStyles
  }));
};

/**
 * Progress (Linear)
 *
 * Progress is used to display the progress status for a task that takes a long
 * time or consists of several steps.
 *
 * It includes accessible attributes to help assistive technologies understand
 * and speak the progress values.
 *
 * @see Docs https://chakra-ui.com/progress
 */
var Progress = function Progress(props) {
  var _styles$track;

  var _omitThemingProps = system.omitThemingProps(props),
      value = _omitThemingProps.value,
      _omitThemingProps$min = _omitThemingProps.min,
      min = _omitThemingProps$min === void 0 ? 0 : _omitThemingProps$min,
      _omitThemingProps$max = _omitThemingProps.max,
      max = _omitThemingProps$max === void 0 ? 100 : _omitThemingProps$max,
      hasStripe = _omitThemingProps.hasStripe,
      isAnimated = _omitThemingProps.isAnimated,
      children = _omitThemingProps.children,
      propBorderRadius = _omitThemingProps.borderRadius,
      isIndeterminate = _omitThemingProps.isIndeterminate,
      ariaLabel = _omitThemingProps["aria-label"],
      ariaLabelledBy = _omitThemingProps["aria-labelledby"],
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded2);

  var styles = system.useMultiStyleConfig("Progress", props);
  var borderRadius = propBorderRadius != null ? propBorderRadius : (_styles$track = styles.track) == null ? void 0 : _styles$track.borderRadius;
  var stripeAnimation = {
    animation: stripe + " 1s linear infinite"
  };
  /**
   * We should not use stripe if it is `indeterminate`
   */

  var shouldAddStripe = !isIndeterminate && hasStripe;
  var shouldAnimateStripe = shouldAddStripe && isAnimated;
  /**
   * Generate styles for stripe and stripe animation
   */

  var css = _extends({}, shouldAnimateStripe && stripeAnimation, isIndeterminate && {
    position: "absolute",
    willChange: "left",
    minWidth: "50%",
    animation: progress + " 1s ease infinite normal none running"
  });

  var trackStyles = _extends({
    overflow: "hidden",
    position: "relative"
  }, styles.track);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    borderRadius: borderRadius,
    __css: trackStyles
  }, rest), /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(ProgressFilledTrack, {
    "aria-label": ariaLabel,
    "aria-labelledby": ariaLabelledBy,
    min: min,
    max: max,
    value: value,
    isIndeterminate: isIndeterminate,
    css: css,
    borderRadius: borderRadius
  }), children));
};

if (utils.__DEV__) {
  Progress.displayName = "Progress";
}

exports.CircularProgress = CircularProgress;
exports.CircularProgressLabel = CircularProgressLabel;
exports.Progress = Progress;
exports.ProgressLabel = ProgressLabel;
