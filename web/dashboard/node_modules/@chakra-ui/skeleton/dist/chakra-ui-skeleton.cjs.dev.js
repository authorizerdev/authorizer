'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var mediaQuery = require('@chakra-ui/media-query');
var system = require('@chakra-ui/system');
var hooks = require('@chakra-ui/hooks');
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

var _excluded = ["startColor", "endColor", "isLoaded", "fadeDuration", "speed", "className"],
    _excluded2 = ["noOfLines", "spacing", "skeletonHeight", "className", "startColor", "endColor", "isLoaded", "fadeDuration", "speed", "children"],
    _excluded3 = ["size"];
var StyledSkeleton = system.chakra("div", {
  baseStyle: {
    boxShadow: "none",
    backgroundClip: "padding-box",
    cursor: "default",
    color: "transparent",
    pointerEvents: "none",
    userSelect: "none",
    "&::before, &::after, *": {
      visibility: "hidden"
    }
  }
});
var fade = system.keyframes({
  from: {
    opacity: 0
  },
  to: {
    opacity: 1
  }
});

var useIsFirstRender = function useIsFirstRender() {
  var isFirstRender = React__namespace.useRef(true);
  React__namespace.useEffect(function () {
    isFirstRender.current = false;
  }, []);
  return isFirstRender.current;
};

var Skeleton = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyleConfig("Skeleton", props);
  var isFirstRender = useIsFirstRender();

  var _omitThemingProps = system.omitThemingProps(props);
      _omitThemingProps.startColor;
      _omitThemingProps.endColor;
      var isLoaded = _omitThemingProps.isLoaded,
      fadeDuration = _omitThemingProps.fadeDuration;
      _omitThemingProps.speed;
      var className = _omitThemingProps.className,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var wasPreviouslyLoaded = hooks.usePrevious(isLoaded);

  var _className = utils.cx("chakra-skeleton", className);

  if (isLoaded) {
    var animation = isFirstRender || wasPreviouslyLoaded ? "none" : fade + " " + fadeDuration + "s";
    return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
      ref: ref,
      className: _className,
      __css: {
        animation: animation
      }
    }, rest));
  }

  return /*#__PURE__*/React__namespace.createElement(StyledSkeleton, _extends({
    ref: ref,
    className: _className
  }, rest, {
    __css: styles
  }));
});
Skeleton.defaultProps = {
  fadeDuration: 0.4,
  speed: 0.8
};

if (utils.__DEV__) {
  Skeleton.displayName = "Skeleton";
}

function range(count) {
  return Array(count).fill(1).map(function (_, index) {
    return index + 1;
  });
}

var defaultNoOfLines = 3;
var SkeletonText = function SkeletonText(props) {
  var _props$noOfLines = props.noOfLines,
      noOfLines = _props$noOfLines === void 0 ? defaultNoOfLines : _props$noOfLines,
      _props$spacing = props.spacing,
      spacing = _props$spacing === void 0 ? "0.5rem" : _props$spacing,
      _props$skeletonHeight = props.skeletonHeight,
      skeletonHeight = _props$skeletonHeight === void 0 ? "0.5rem" : _props$skeletonHeight,
      className = props.className,
      startColor = props.startColor,
      endColor = props.endColor,
      isLoaded = props.isLoaded,
      fadeDuration = props.fadeDuration,
      speed = props.speed,
      children = props.children,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);

  var noOfLinesValue = mediaQuery.useBreakpointValue(typeof noOfLines === "number" ? [noOfLines] : noOfLines) || defaultNoOfLines;
  var numbers = range(noOfLinesValue);

  var getWidth = function getWidth(index) {
    if (noOfLinesValue > 1) {
      return index === numbers.length ? "80%" : "100%";
    }

    return "100%";
  };

  var _className = utils.cx("chakra-skeleton__group", className);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    className: _className
  }, rest), numbers.map(function (number, index) {
    if (isLoaded && index > 0) {
      // skip other lines
      return null;
    }

    var sizeProps = isLoaded ? null : {
      mb: number === numbers.length ? "0" : spacing,
      width: getWidth(number),
      height: skeletonHeight
    };
    return /*#__PURE__*/React__namespace.createElement(Skeleton, _extends({
      key: numbers.length.toString() + number,
      startColor: startColor,
      endColor: endColor,
      isLoaded: isLoaded,
      fadeDuration: fadeDuration,
      speed: speed
    }, sizeProps), // allows animating the children
    index === 0 ? children : undefined);
  }));
};

if (utils.__DEV__) {
  SkeletonText.displayName = "SkeletonText";
}

var SkeletonCircle = function SkeletonCircle(_ref) {
  var _ref$size = _ref.size,
      size = _ref$size === void 0 ? "2rem" : _ref$size,
      rest = _objectWithoutPropertiesLoose(_ref, _excluded3);

  return /*#__PURE__*/React__namespace.createElement(Skeleton, _extends({
    borderRadius: "full",
    boxSize: size
  }, rest));
};

if (utils.__DEV__) {
  SkeletonCircle.displayName = "SkeletonCircle";
}

exports.Skeleton = Skeleton;
exports.SkeletonCircle = SkeletonCircle;
exports.SkeletonText = SkeletonText;
