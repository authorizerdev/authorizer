import { mergeRefs, createContext } from '@chakra-ui/react-utils';
import { forwardRef, useMultiStyleConfig, omitThemingProps, useTheme, StylesProvider, chakra, useStyles } from '@chakra-ui/system';
import { clampValue, valueToPercent, percentToValue, roundValueToStep, normalizeEventKey, focus, ariaAttr, dataAttr, callAllHandlers, __DEV__, cx, getBox } from '@chakra-ui/utils';
import * as React from 'react';
import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { useCallbackRef, useControllableState, useBoolean, useLatestRef, useId, useUpdateEffect, usePanGesture, useIds, useDimensions } from '@chakra-ui/hooks';

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

function getIds(id) {
  return {
    root: "slider-root-" + id,
    getThumb: function getThumb(i) {
      return "slider-thumb-" + id + "-" + i;
    },
    getInput: function getInput(i) {
      return "slider-input-" + id + "-" + i;
    },
    track: "slider-track-" + id,
    innerTrack: "slider-filled-track-" + id,
    getMarker: function getMarker(i) {
      return "slider-marker-" + id + "-" + i;
    },
    output: "slider-output-" + id
  };
}
function orient$1(options) {
  var orientation = options.orientation,
      vertical = options.vertical,
      horizontal = options.horizontal;
  return orientation === "vertical" ? vertical : horizontal;
}
var zeroRect = {
  width: 0,
  height: 0
};
function getStyles(options) {
  var orientation = options.orientation,
      thumbPercents = options.thumbPercents,
      thumbRects = options.thumbRects,
      isReversed = options.isReversed;

  var getThumbStyle = function getThumbStyle(i) {
    return _extends({
      position: "absolute",
      userSelect: "none",
      WebkitUserSelect: "none",
      MozUserSelect: "none",
      msUserSelect: "none",
      touchAction: "none"
    }, orient$1({
      orientation: orientation,
      vertical: {
        bottom: "calc(" + thumbPercents[i] + "% - " + thumbRects[i].height / 2 + "px)"
      },
      horizontal: {
        left: "calc(" + thumbPercents[i] + "% - " + thumbRects[i].width / 2 + "px)"
      }
    }));
  };

  var size = orientation === "vertical" ? thumbRects.reduce(function (a, b) {
    return a.height > b.height ? a : b;
  }, zeroRect) : thumbRects.reduce(function (a, b) {
    return a.width > b.width ? a : b;
  }, zeroRect);

  var rootStyle = _extends({
    position: "relative",
    touchAction: "none",
    WebkitTapHighlightColor: "rgba(0,0,0,0)",
    userSelect: "none",
    outline: 0
  }, orient$1({
    orientation: orientation,
    vertical: {
      paddingLeft: size.width / 2,
      paddingRight: size.width / 2
    },
    horizontal: {
      paddingTop: size.height / 2,
      paddingBottom: size.height / 2
    }
  }));

  var trackStyle = _extends({
    position: "absolute"
  }, orient$1({
    orientation: orientation,
    vertical: {
      left: "50%",
      transform: "translateX(-50%)",
      height: "100%"
    },
    horizontal: {
      top: "50%",
      transform: "translateY(-50%)",
      width: "100%"
    }
  }));

  var isSingleThumb = thumbPercents.length === 1;
  var fallback = [0, isReversed ? 100 - thumbPercents[0] : thumbPercents[0]];
  var range = isSingleThumb ? fallback : thumbPercents;
  var start = range[0];

  if (!isSingleThumb && isReversed) {
    start = 100 - start;
  }

  var percent = Math.abs(range[range.length - 1] - range[0]);

  var innerTrackStyle = _extends({}, trackStyle, orient$1({
    orientation: orientation,
    vertical: isReversed ? {
      height: percent + "%",
      top: start + "%"
    } : {
      height: percent + "%",
      bottom: start + "%"
    },
    horizontal: isReversed ? {
      width: percent + "%",
      right: start + "%"
    } : {
      width: percent + "%",
      left: start + "%"
    }
  }));

  return {
    trackStyle: trackStyle,
    innerTrackStyle: innerTrackStyle,
    rootStyle: rootStyle,
    getThumbStyle: getThumbStyle
  };
}
function getIsReversed(options) {
  var isReversed = options.isReversed,
      direction = options.direction,
      orientation = options.orientation;

  if (direction === "ltr" || orientation === "vertical") {
    return isReversed;
  } // only flip for horizontal RTL
  // if isReserved 🔜  otherwise  🔚


  return !isReversed;
}

var _excluded$3 = ["min", "max", "onChange", "value", "defaultValue", "isReversed", "direction", "orientation", "id", "isDisabled", "isReadOnly", "onChangeStart", "onChangeEnd", "step", "getAriaValueText", "aria-valuetext", "aria-label", "aria-labelledby", "name", "focusThumbOnChange", "minStepsBetweenThumbs"],
    _excluded2 = ["index"],
    _excluded3 = ["value"],
    _excluded4 = ["index"];

/**
 * React hook that implements an accessible range slider.
 *
 * It is an alternative to `<input type="range" />`, and returns
 * prop getters for the component parts
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices-1.1/#slider
 */
function useRangeSlider(props) {
  var _props$min = props.min,
      min = _props$min === void 0 ? 0 : _props$min,
      _props$max = props.max,
      max = _props$max === void 0 ? 100 : _props$max,
      onChange = props.onChange,
      valueProp = props.value,
      defaultValue = props.defaultValue,
      isReversedProp = props.isReversed,
      _props$direction = props.direction,
      direction = _props$direction === void 0 ? "ltr" : _props$direction,
      _props$orientation = props.orientation,
      orientation = _props$orientation === void 0 ? "horizontal" : _props$orientation,
      idProp = props.id,
      isDisabled = props.isDisabled,
      isReadOnly = props.isReadOnly,
      onChangeStartProp = props.onChangeStart,
      onChangeEndProp = props.onChangeEnd,
      _props$step = props.step,
      step = _props$step === void 0 ? 1 : _props$step,
      getAriaValueTextProp = props.getAriaValueText,
      ariaValueText = props["aria-valuetext"],
      ariaLabel = props["aria-label"],
      ariaLabelledBy = props["aria-labelledby"],
      name = props.name,
      _props$focusThumbOnCh = props.focusThumbOnChange,
      focusThumbOnChange = _props$focusThumbOnCh === void 0 ? true : _props$focusThumbOnCh,
      _props$minStepsBetwee = props.minStepsBetweenThumbs,
      minStepsBetweenThumbs = _props$minStepsBetwee === void 0 ? 0 : _props$minStepsBetwee,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded$3);

  var onChangeStart = useCallbackRef(onChangeStartProp);
  var onChangeEnd = useCallbackRef(onChangeEndProp);
  var getAriaValueText = useCallbackRef(getAriaValueTextProp);
  var isReversed = getIsReversed({
    isReversed: isReversedProp,
    direction: direction,
    orientation: orientation
  });

  var _useControllableState = useControllableState({
    value: valueProp,
    defaultValue: defaultValue != null ? defaultValue : [25, 75],
    onChange: onChange
  }),
      valueState = _useControllableState[0],
      setValue = _useControllableState[1];

  if (!Array.isArray(valueState)) {
    throw new TypeError("[range-slider] You passed an invalid value for `value` or `defaultValue`, expected `Array` but got `" + typeof valueState + "`");
  }

  var _useBoolean = useBoolean(),
      isDragging = _useBoolean[0],
      setDragging = _useBoolean[1];

  var _useBoolean2 = useBoolean(),
      isFocused = _useBoolean2[0],
      setFocused = _useBoolean2[1];

  var _useState = useState(-1),
      activeIndex = _useState[0],
      setActiveIndex = _useState[1];

  var eventSourceRef = useRef(null);
  var isInteractive = !(isDisabled || isReadOnly);
  var initialValue = useRef(valueState);
  var value = valueState.map(function (val) {
    return clampValue(val, min, max);
  });
  var valueRef = useLatestRef(value);
  var spacing = minStepsBetweenThumbs * step;
  var valueBounds = getValueBounds(value, min, max, spacing);
  var reversedValue = value.map(function (val) {
    return max - val + min;
  });
  var thumbValues = isReversed ? reversedValue : value;
  var thumbPercents = thumbValues.map(function (val) {
    return valueToPercent(val, min, max);
  });
  var isVertical = orientation === "vertical";

  var _useState2 = useState(Array.from({
    length: value.length
  }).map(function () {
    return {
      width: 0,
      height: 0
    };
  })),
      thumbRects = _useState2[0],
      setThumbRects = _useState2[1];

  useEffect(function () {
    var _rootRef$current;

    if (!rootRef.current) return;
    var thumbs = Array.from((_rootRef$current = rootRef.current) == null ? void 0 : _rootRef$current.querySelectorAll("[role=slider]"));
    var rects = thumbs.map(function (el) {
      return {
        width: el.offsetWidth,
        height: el.offsetHeight
      };
    });
    if (rects.length) setThumbRects(rects);
  }, []);
  /**
   * Let's keep a reference to the slider track and thumb
   */

  var trackRef = useRef(null);
  var rootRef = useRef(null);
  var uuid = useId(idProp);
  var ids = getIds(uuid);
  var getValueFromPointer = useCallback(function (event) {
    var _event$touches$, _event$touches;

    if (!trackRef.current) return;
    eventSourceRef.current = "pointer";
    var rect = trackRef.current.getBoundingClientRect();

    var _ref = (_event$touches$ = (_event$touches = event.touches) == null ? void 0 : _event$touches[0]) != null ? _event$touches$ : event,
        clientX = _ref.clientX,
        clientY = _ref.clientY;

    var diff = isVertical ? rect.bottom - clientY : clientX - rect.left;
    var length = isVertical ? rect.height : rect.width;
    var percent = diff / length;
    if (isReversed) percent = 1 - percent;
    return percentToValue(percent, min, max);
  }, [isVertical, isReversed, max, min]);
  var tenSteps = (max - min) / 10;
  var oneStep = step || (max - min) / 100;
  var actions = useMemo(function () {
    return {
      setValueAtIndex: function setValueAtIndex(index, val) {
        if (!isInteractive) return;
        var bounds = valueBounds[index];
        val = parseFloat(roundValueToStep(val, bounds.min, oneStep));
        val = clampValue(val, bounds.min, bounds.max);
        var next = [].concat(value);
        next[index] = val;
        setValue(next);
      },
      setActiveIndex: setActiveIndex,
      stepUp: function stepUp(index, step) {
        if (step === void 0) {
          step = oneStep;
        }

        var valueAtIndex = value[index];
        var next = isReversed ? valueAtIndex - step : valueAtIndex + step;
        actions.setValueAtIndex(index, next);
      },
      stepDown: function stepDown(index, step) {
        if (step === void 0) {
          step = oneStep;
        }

        var valueAtIndex = value[index];
        var next = isReversed ? valueAtIndex + step : valueAtIndex - step;
        actions.setValueAtIndex(index, next);
      },
      reset: function reset() {
        return setValue(initialValue.current);
      }
    };
  }, [oneStep, value, isReversed, setValue, isInteractive, valueBounds]);
  /**
   * Keyboard interaction to ensure users can operate
   * the slider using only their keyboard.
   */

  var onKeyDown = useCallback(function (event) {
    var eventKey = normalizeEventKey(event);
    var keyMap = {
      ArrowRight: function ArrowRight() {
        return actions.stepUp(activeIndex);
      },
      ArrowUp: function ArrowUp() {
        return actions.stepUp(activeIndex);
      },
      ArrowLeft: function ArrowLeft() {
        return actions.stepDown(activeIndex);
      },
      ArrowDown: function ArrowDown() {
        return actions.stepDown(activeIndex);
      },
      PageUp: function PageUp() {
        return actions.stepUp(activeIndex, tenSteps);
      },
      PageDown: function PageDown() {
        return actions.stepDown(activeIndex, tenSteps);
      },
      Home: function Home() {
        var value = valueBounds[activeIndex].min;
        actions.setValueAtIndex(activeIndex, value);
      },
      End: function End() {
        var value = valueBounds[activeIndex].max;
        actions.setValueAtIndex(activeIndex, value);
      }
    };
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      event.stopPropagation();
      action(event);
      eventSourceRef.current = "keyboard";
    }
  }, [actions, activeIndex, tenSteps, valueBounds]);
  /**
   * Compute styles for all component parts.
   */

  var _useMemo = useMemo(function () {
    return getStyles({
      isReversed: isReversed,
      orientation: orientation,
      thumbRects: thumbRects,
      thumbPercents: thumbPercents
    });
  }, [isReversed, orientation, thumbPercents, thumbRects]),
      getThumbStyle = _useMemo.getThumbStyle,
      rootStyle = _useMemo.rootStyle,
      trackStyle = _useMemo.trackStyle,
      innerTrackStyle = _useMemo.innerTrackStyle;

  var focusThumb = useCallback(function (index) {
    var idx = index != null ? index : activeIndex;

    if (idx !== -1 && focusThumbOnChange) {
      var _rootRef$current2;

      var id = ids.getThumb(idx);
      var thumb = (_rootRef$current2 = rootRef.current) == null ? void 0 : _rootRef$current2.ownerDocument.getElementById(id);

      if (thumb) {
        setTimeout(function () {
          return focus(thumb);
        });
      }
    }
  }, [focusThumbOnChange, activeIndex, ids]);
  useUpdateEffect(function () {
    if (eventSourceRef.current === "keyboard") {
      onChangeEnd == null ? void 0 : onChangeEnd(valueRef.current);
    }
  }, [value, onChangeEnd]);

  var _onPanSessionStart = function onPanSessionStart(event) {
    var pointValue = getValueFromPointer(event) || 0;
    var distances = value.map(function (val) {
      return Math.abs(val - pointValue);
    });
    var closest = Math.min.apply(Math, distances);
    var index = distances.indexOf(closest); // check if the clicked thumb is stacked by checking if there are multiple
    // thumbs at the same distance

    var thumbsAtPosition = distances.filter(function (distance) {
      return distance === closest;
    });
    var isThumbStacked = thumbsAtPosition.length > 1; // when two thumbs are stacked and the user clicks at a point larger than
    // their values, pick the last thumb with the greatest index

    if (isThumbStacked && pointValue > value[index]) {
      index = thumbsAtPosition.length - 1;
    }

    setActiveIndex(index);
    actions.setValueAtIndex(index, pointValue);
    focusThumb(index);
  };

  var _onPan = function onPan(event) {
    if (activeIndex == -1) return;
    var pointValue = getValueFromPointer(event) || 0;
    setActiveIndex(activeIndex);
    actions.setValueAtIndex(activeIndex, pointValue);
    focusThumb(activeIndex);
  };

  usePanGesture(rootRef, {
    onPanSessionStart: function onPanSessionStart(event) {
      if (!isInteractive) return;
      setDragging.on();

      _onPanSessionStart(event);

      onChangeStart == null ? void 0 : onChangeStart(valueRef.current);
    },
    onPanSessionEnd: function onPanSessionEnd() {
      if (!isInteractive) return;
      setDragging.off();
      onChangeEnd == null ? void 0 : onChangeEnd(valueRef.current);
    },
    onPan: function onPan(event) {
      if (!isInteractive) return;

      _onPan(event);
    }
  });
  var getRootProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, htmlProps, {
      id: ids.root,
      ref: mergeRefs(ref, rootRef),
      tabIndex: -1,
      "aria-disabled": ariaAttr(isDisabled),
      "data-focused": dataAttr(isFocused),
      style: _extends({}, props.style, rootStyle)
    });
  }, [htmlProps, isDisabled, isFocused, rootStyle, ids]);
  var getTrackProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: mergeRefs(ref, trackRef),
      id: ids.track,
      "data-disabled": dataAttr(isDisabled),
      style: _extends({}, props.style, trackStyle)
    });
  }, [isDisabled, trackStyle, ids]);
  var getInnerTrackProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      id: ids.innerTrack,
      style: _extends({}, props.style, innerTrackStyle)
    });
  }, [innerTrackStyle, ids]);
  var getThumbProps = useCallback(function (props, ref) {
    var _getAriaValueText;

    if (ref === void 0) {
      ref = null;
    }

    var index = props.index,
        rest = _objectWithoutPropertiesLoose(props, _excluded2);

    var _value = value[index];

    if (_value == null) {
      throw new TypeError("[range-slider > thumb] Cannot find value at index `" + index + "`. The `value` or `defaultValue` length is : " + value.length);
    }

    var bounds = valueBounds[index];
    return _extends({}, rest, {
      ref: ref,
      role: "slider",
      tabIndex: isInteractive ? 0 : undefined,
      id: ids.getThumb(index),
      "data-active": dataAttr(isDragging && activeIndex === index),
      "aria-valuetext": (_getAriaValueText = getAriaValueText == null ? void 0 : getAriaValueText(_value)) != null ? _getAriaValueText : ariaValueText == null ? void 0 : ariaValueText[index],
      "aria-valuemin": bounds.min,
      "aria-valuemax": bounds.max,
      "aria-valuenow": _value,
      "aria-orientation": orientation,
      "aria-disabled": ariaAttr(isDisabled),
      "aria-readonly": ariaAttr(isReadOnly),
      "aria-label": ariaLabel == null ? void 0 : ariaLabel[index],
      "aria-labelledby": ariaLabel != null && ariaLabel[index] ? undefined : ariaLabelledBy == null ? void 0 : ariaLabelledBy[index],
      style: _extends({}, props.style, getThumbStyle(index)),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown),
      onFocus: callAllHandlers(props.onFocus, function () {
        setFocused.on();
        setActiveIndex(index);
      }),
      onBlur: callAllHandlers(props.onBlur, function () {
        setFocused.off();
        setActiveIndex(-1);
      })
    });
  }, [ids, value, valueBounds, isInteractive, isDragging, activeIndex, getAriaValueText, ariaValueText, orientation, isDisabled, isReadOnly, ariaLabel, ariaLabelledBy, getThumbStyle, onKeyDown, setFocused]);
  var getOutputProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      id: ids.output,
      htmlFor: value.map(function (v, i) {
        return ids.getThumb(i);
      }).join(" "),
      "aria-live": "off"
    });
  }, [ids, value]);
  var getMarkerProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var _props = props,
        v = _props.value,
        rest = _objectWithoutPropertiesLoose(_props, _excluded3);

    var isInRange = !(v < min || v > max);
    var isHighlighted = v >= value[0] && v <= value[value.length - 1];
    var percent = valueToPercent(v, min, max);
    percent = isReversed ? 100 - percent : percent;

    var markerStyle = _extends({
      position: "absolute",
      pointerEvents: "none"
    }, orient$1({
      orientation: orientation,
      vertical: {
        bottom: percent + "%"
      },
      horizontal: {
        left: percent + "%"
      }
    }));

    return _extends({}, rest, {
      ref: ref,
      id: ids.getMarker(props.value),
      role: "presentation",
      "aria-hidden": true,
      "data-disabled": dataAttr(isDisabled),
      "data-invalid": dataAttr(!isInRange),
      "data-highlighted": dataAttr(isHighlighted),
      style: _extends({}, props.style, markerStyle)
    });
  }, [isDisabled, isReversed, max, min, orientation, value, ids]);
  var getInputProps = useCallback(function (props, ref) {
    if (ref === void 0) {
      ref = null;
    }

    var index = props.index,
        rest = _objectWithoutPropertiesLoose(props, _excluded4);

    return _extends({}, rest, {
      ref: ref,
      id: ids.getInput(index),
      type: "hidden",
      value: value[index],
      name: Array.isArray(name) ? name[index] : name + "-" + index
    });
  }, [name, value, ids]);
  return {
    state: {
      value: value,
      isFocused: isFocused,
      isDragging: isDragging,
      getThumbPercent: function getThumbPercent(i) {
        return thumbPercents[i];
      },
      getThumbMinValue: function getThumbMinValue(i) {
        return valueBounds[i].min;
      },
      getThumbMaxValue: function getThumbMaxValue(i) {
        return valueBounds[i].max;
      }
    },
    actions: actions,
    getRootProps: getRootProps,
    getTrackProps: getTrackProps,
    getInnerTrackProps: getInnerTrackProps,
    getThumbProps: getThumbProps,
    getMarkerProps: getMarkerProps,
    getInputProps: getInputProps,
    getOutputProps: getOutputProps
  };
}

var getValueBounds = function getValueBounds(arr, min, max, spacing) {
  return arr.map(function (v, i) {
    var _min = i === 0 ? min : arr[i - 1] + spacing;

    var _max = i === arr.length - 1 ? max : arr[i + 1] - spacing;

    return {
      min: _min,
      max: _max
    };
  });
};

var _excluded$2 = ["getRootProps"];

var _createContext$1 = createContext({
  name: "SliderContext",
  errorMessage: "useSliderContext: `context` is undefined. Seems you forgot to wrap all slider components within <RangeSlider />"
}),
    RangeSliderProvider = _createContext$1[0],
    useRangeSliderContext = _createContext$1[1];

/**
 * The Slider is used to allow users to make selections from a range of values.
 * It provides context and functionality for all slider components
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices/#slider
 */
var RangeSlider = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Slider", props);
  var ownProps = omitThemingProps(props);

  var _useTheme = useTheme(),
      direction = _useTheme.direction;

  ownProps.direction = direction;

  var _useRangeSlider = useRangeSlider(ownProps),
      getRootProps = _useRangeSlider.getRootProps,
      context = _objectWithoutPropertiesLoose(_useRangeSlider, _excluded$2);

  var ctx = React.useMemo(function () {
    return _extends({}, context, {
      name: props.name
    });
  }, [context, props.name]);
  return /*#__PURE__*/React.createElement(RangeSliderProvider, {
    value: ctx
  }, /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({}, getRootProps({}, ref), {
    className: "chakra-slider",
    __css: styles.container
  }), props.children)));
});
RangeSlider.defaultProps = {
  orientation: "horizontal"
};

if (__DEV__) {
  RangeSlider.displayName = "RangeSlider";
}

/**
 * Slider component that acts as the handle used to select predefined
 * values by dragging its handle along the track
 */
var RangeSliderThumb = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useRangeSliderContex = useRangeSliderContext(),
      getThumbProps = _useRangeSliderContex.getThumbProps,
      getInputProps = _useRangeSliderContex.getInputProps,
      name = _useRangeSliderContex.name;

  var styles = useStyles();
  var thumbProps = getThumbProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, thumbProps, {
    className: cx("chakra-slider__thumb", props.className),
    __css: styles.thumb
  }), thumbProps.children, name && /*#__PURE__*/React.createElement("input", getInputProps({
    index: props.index
  })));
});

if (__DEV__) {
  RangeSliderThumb.displayName = "RangeSliderThumb";
}

var RangeSliderTrack = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useRangeSliderContex2 = useRangeSliderContext(),
      getTrackProps = _useRangeSliderContex2.getTrackProps;

  var styles = useStyles();
  var trackProps = getTrackProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, trackProps, {
    className: cx("chakra-slider__track", props.className),
    __css: styles.track,
    "data-testid": "chakra-range-slider-track"
  }));
});

if (__DEV__) {
  RangeSliderTrack.displayName = "RangeSliderTrack";
}

var RangeSliderFilledTrack = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useRangeSliderContex3 = useRangeSliderContext(),
      getInnerTrackProps = _useRangeSliderContex3.getInnerTrackProps;

  var styles = useStyles();
  var trackProps = getInnerTrackProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, trackProps, {
    className: "chakra-slider__filled-track",
    __css: styles.filledTrack
  }));
});

if (__DEV__) {
  RangeSliderFilledTrack.displayName = "RangeSliderFilledTrack";
}

/**
 * SliderMark is used to provide names for specific Slider
 * values by defining labels or markers along the track.
 *
 * @see Docs https://chakra-ui.com/slider
 */
var RangeSliderMark = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useRangeSliderContex4 = useRangeSliderContext(),
      getMarkerProps = _useRangeSliderContex4.getMarkerProps;

  var markProps = getMarkerProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, markProps, {
    className: cx("chakra-slider__marker", props.className)
  }));
});

if (__DEV__) {
  RangeSliderMark.displayName = "RangeSliderMark";
}

var _excluded$1 = ["min", "max", "onChange", "value", "defaultValue", "isReversed", "direction", "orientation", "id", "isDisabled", "isReadOnly", "onChangeStart", "onChangeEnd", "step", "getAriaValueText", "aria-valuetext", "aria-label", "aria-labelledby", "name", "focusThumbOnChange"];

/**
 * React hook that implements an accessible range slider.
 *
 * It is an alternative to `<input type="range" />`, and returns
 * prop getters for the component parts
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices-1.1/#slider
 */
function useSlider(props) {
  var _getAriaValueText;

  var _props$min = props.min,
      min = _props$min === void 0 ? 0 : _props$min,
      _props$max = props.max,
      max = _props$max === void 0 ? 100 : _props$max,
      onChange = props.onChange,
      valueProp = props.value,
      defaultValue = props.defaultValue,
      isReversedProp = props.isReversed,
      _props$direction = props.direction,
      direction = _props$direction === void 0 ? "ltr" : _props$direction,
      _props$orientation = props.orientation,
      orientation = _props$orientation === void 0 ? "horizontal" : _props$orientation,
      idProp = props.id,
      isDisabled = props.isDisabled,
      isReadOnly = props.isReadOnly,
      onChangeStartProp = props.onChangeStart,
      onChangeEndProp = props.onChangeEnd,
      _props$step = props.step,
      step = _props$step === void 0 ? 1 : _props$step,
      getAriaValueTextProp = props.getAriaValueText,
      ariaValueText = props["aria-valuetext"],
      ariaLabel = props["aria-label"],
      ariaLabelledBy = props["aria-labelledby"],
      name = props.name,
      _props$focusThumbOnCh = props.focusThumbOnChange,
      focusThumbOnChange = _props$focusThumbOnCh === void 0 ? true : _props$focusThumbOnCh,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded$1);

  var onChangeStart = useCallbackRef(onChangeStartProp);
  var onChangeEnd = useCallbackRef(onChangeEndProp);
  var getAriaValueText = useCallbackRef(getAriaValueTextProp);
  var isReversed = getIsReversed({
    isReversed: isReversedProp,
    direction: direction,
    orientation: orientation
  });
  /**
   * Enable the slider handle controlled and uncontrolled scenarios
   */

  var _useControllableState = useControllableState({
    value: valueProp,
    defaultValue: defaultValue != null ? defaultValue : getDefaultValue(min, max),
    onChange: onChange
  }),
      computedValue = _useControllableState[0],
      setValue = _useControllableState[1];

  var _useBoolean = useBoolean(),
      isDragging = _useBoolean[0],
      setDragging = _useBoolean[1];

  var _useBoolean2 = useBoolean(),
      isFocused = _useBoolean2[0],
      setFocused = _useBoolean2[1];

  var eventSourceRef = useRef(null);
  var isInteractive = !(isDisabled || isReadOnly);
  /**
   * Constrain the value because it can't be less than min
   * or greater than max
   */

  var value = clampValue(computedValue, min, max);
  var valueRef = useLatestRef(value);
  var prevRef = useRef(valueRef.current);
  var reversedValue = max - value + min;
  var trackValue = isReversed ? reversedValue : value;
  var thumbPercent = valueToPercent(trackValue, min, max);
  var isVertical = orientation === "vertical";
  /**
   * Let's keep a reference to the slider track and thumb
   */

  var trackRef = useRef(null);
  var thumbRef = useRef(null);
  var rootRef = useRef(null);
  /**
   * Generate unique ids for component parts
   */

  var _useIds = useIds(idProp, "slider-thumb", "slider-track"),
      thumbId = _useIds[0],
      trackId = _useIds[1];
  /**
   * Get relative value of slider from the event by tracking
   * how far you clicked within the track to determine the value
   *
   * @todo - Refactor this later on to use info from pan session
   */


  var getValueFromPointer = useCallback(function (event) {
    var _event$touches$, _event$touches;

    if (!trackRef.current) return;
    eventSourceRef.current = "pointer";
    var trackRect = getBox(trackRef.current).borderBox;

    var _ref = (_event$touches$ = (_event$touches = event.touches) == null ? void 0 : _event$touches[0]) != null ? _event$touches$ : event,
        clientX = _ref.clientX,
        clientY = _ref.clientY;

    var diff = isVertical ? trackRect.bottom - clientY : clientX - trackRect.left;
    var length = isVertical ? trackRect.height : trackRect.width;
    var percent = diff / length;

    if (isReversed) {
      percent = 1 - percent;
    }

    var nextValue = percentToValue(percent, min, max);

    if (step) {
      nextValue = parseFloat(roundValueToStep(nextValue, min, step));
    }

    nextValue = clampValue(nextValue, min, max);
    return nextValue;
  }, [isVertical, isReversed, max, min, step]);
  var tenSteps = (max - min) / 10;
  var oneStep = step || (max - min) / 100;
  var constrain = useCallback(function (value) {
    if (!isInteractive) return;
    value = parseFloat(roundValueToStep(value, min, oneStep));
    value = clampValue(value, min, max);
    setValue(value);
  }, [oneStep, max, min, setValue, isInteractive]);
  var actions = useMemo(function () {
    return {
      stepUp: function stepUp(step) {
        if (step === void 0) {
          step = oneStep;
        }

        var next = isReversed ? value - step : value + step;
        constrain(next);
      },
      stepDown: function stepDown(step) {
        if (step === void 0) {
          step = oneStep;
        }

        var next = isReversed ? value + step : value - step;
        constrain(next);
      },
      reset: function reset() {
        return constrain(defaultValue || 0);
      },
      stepTo: function stepTo(value) {
        return constrain(value);
      }
    };
  }, [constrain, isReversed, value, oneStep, defaultValue]);
  /**
   * Keyboard interaction to ensure users can operate
   * the slider using only their keyboard.
   */

  var onKeyDown = useCallback(function (event) {
    var eventKey = normalizeEventKey(event);
    var keyMap = {
      ArrowRight: function ArrowRight() {
        return actions.stepUp();
      },
      ArrowUp: function ArrowUp() {
        return actions.stepUp();
      },
      ArrowLeft: function ArrowLeft() {
        return actions.stepDown();
      },
      ArrowDown: function ArrowDown() {
        return actions.stepDown();
      },
      PageUp: function PageUp() {
        return actions.stepUp(tenSteps);
      },
      PageDown: function PageDown() {
        return actions.stepDown(tenSteps);
      },
      Home: function Home() {
        return constrain(min);
      },
      End: function End() {
        return constrain(max);
      }
    };
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      event.stopPropagation();
      action(event);
      eventSourceRef.current = "keyboard";
    }
  }, [actions, constrain, max, min, tenSteps]);
  /**
   * ARIA (Optional): To define a human readable representation of the value,
   * we allow users pass aria-valuetext.
   */

  var valueText = (_getAriaValueText = getAriaValueText == null ? void 0 : getAriaValueText(value)) != null ? _getAriaValueText : ariaValueText;
  /**
   * Measure the dimensions of the thumb so
   * we can center it within the track properly
   */

  var thumbBoxModel = useDimensions(thumbRef);
  /**
   * Compute styles for all component parts.
   */

  var _useMemo = useMemo(function () {
    var _thumbBoxModel$border;

    var thumbRect = (_thumbBoxModel$border = thumbBoxModel == null ? void 0 : thumbBoxModel.borderBox) != null ? _thumbBoxModel$border : {
      width: 0,
      height: 0
    };
    return getStyles({
      isReversed: isReversed,
      orientation: orientation,
      thumbRects: [thumbRect],
      thumbPercents: [thumbPercent]
    });
  }, [isReversed, orientation, thumbBoxModel == null ? void 0 : thumbBoxModel.borderBox, thumbPercent]),
      getThumbStyle = _useMemo.getThumbStyle,
      rootStyle = _useMemo.rootStyle,
      trackStyle = _useMemo.trackStyle,
      innerTrackStyle = _useMemo.innerTrackStyle;

  var focusThumb = useCallback(function () {
    if (thumbRef.current && focusThumbOnChange) {
      setTimeout(function () {
        return focus(thumbRef.current);
      });
    }
  }, [focusThumbOnChange]);
  useUpdateEffect(function () {
    focusThumb();

    if (eventSourceRef.current === "keyboard") {
      onChangeEnd == null ? void 0 : onChangeEnd(valueRef.current);
    }
  }, [value, onChangeEnd]);

  var setValueFromPointer = function setValueFromPointer(event) {
    var nextValue = getValueFromPointer(event);

    if (nextValue != null && nextValue !== valueRef.current) {
      setValue(nextValue);
    }
  };

  usePanGesture(rootRef, {
    onPanSessionStart: function onPanSessionStart(event) {
      if (!isInteractive) return;
      setDragging.on();
      focusThumb();
      setValueFromPointer(event);
      onChangeStart == null ? void 0 : onChangeStart(valueRef.current);
    },
    onPanSessionEnd: function onPanSessionEnd() {
      if (!isInteractive) return;
      setDragging.off();
      onChangeEnd == null ? void 0 : onChangeEnd(valueRef.current);
      prevRef.current = valueRef.current;
    },
    onPan: function onPan(event) {
      if (!isInteractive) return;
      setValueFromPointer(event);
    }
  });
  var getRootProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, htmlProps, {
      ref: mergeRefs(ref, rootRef),
      tabIndex: -1,
      "aria-disabled": ariaAttr(isDisabled),
      "data-focused": dataAttr(isFocused),
      style: _extends({}, props.style, rootStyle)
    });
  }, [htmlProps, isDisabled, isFocused, rootStyle]);
  var getTrackProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: mergeRefs(ref, trackRef),
      id: trackId,
      "data-disabled": dataAttr(isDisabled),
      style: _extends({}, props.style, trackStyle)
    });
  }, [isDisabled, trackId, trackStyle]);
  var getInnerTrackProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      style: _extends({}, props.style, innerTrackStyle)
    });
  }, [innerTrackStyle]);
  var getThumbProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: mergeRefs(ref, thumbRef),
      role: "slider",
      tabIndex: isInteractive ? 0 : undefined,
      id: thumbId,
      "data-active": dataAttr(isDragging),
      "aria-valuetext": valueText,
      "aria-valuemin": min,
      "aria-valuemax": max,
      "aria-valuenow": value,
      "aria-orientation": orientation,
      "aria-disabled": ariaAttr(isDisabled),
      "aria-readonly": ariaAttr(isReadOnly),
      "aria-label": ariaLabel,
      "aria-labelledby": ariaLabel ? undefined : ariaLabelledBy,
      style: _extends({}, props.style, getThumbStyle(0)),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown),
      onFocus: callAllHandlers(props.onFocus, setFocused.on),
      onBlur: callAllHandlers(props.onBlur, setFocused.off)
    });
  }, [isInteractive, thumbId, isDragging, valueText, min, max, value, orientation, isDisabled, isReadOnly, ariaLabel, ariaLabelledBy, getThumbStyle, onKeyDown, setFocused.on, setFocused.off]);
  var getMarkerProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var isInRange = !(props.value < min || props.value > max);
    var isHighlighted = value >= props.value;
    var markerPercent = valueToPercent(props.value, min, max);

    var markerStyle = _extends({
      position: "absolute",
      pointerEvents: "none"
    }, orient({
      orientation: orientation,
      vertical: {
        bottom: isReversed ? 100 - markerPercent + "%" : markerPercent + "%"
      },
      horizontal: {
        left: isReversed ? 100 - markerPercent + "%" : markerPercent + "%"
      }
    }));

    return _extends({}, props, {
      ref: ref,
      role: "presentation",
      "aria-hidden": true,
      "data-disabled": dataAttr(isDisabled),
      "data-invalid": dataAttr(!isInRange),
      "data-highlighted": dataAttr(isHighlighted),
      style: _extends({}, props.style, markerStyle)
    });
  }, [isDisabled, isReversed, max, min, orientation, value]);
  var getInputProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      type: "hidden",
      value: value,
      name: name
    });
  }, [name, value]);
  return {
    state: {
      value: value,
      isFocused: isFocused,
      isDragging: isDragging
    },
    actions: actions,
    getRootProps: getRootProps,
    getTrackProps: getTrackProps,
    getInnerTrackProps: getInnerTrackProps,
    getThumbProps: getThumbProps,
    getMarkerProps: getMarkerProps,
    getInputProps: getInputProps
  };
}

function orient(options) {
  var orientation = options.orientation,
      vertical = options.vertical,
      horizontal = options.horizontal;
  return orientation === "vertical" ? vertical : horizontal;
}
/**
 * The browser <input type="range" /> calculates
 * the default value of a slider by using mid-point
 * between the min and the max.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input/range
 */


function getDefaultValue(min, max) {
  return max < min ? min : min + (max - min) / 2;
}

var _excluded = ["getInputProps", "getRootProps"];

var _createContext = createContext({
  name: "SliderContext",
  errorMessage: "useSliderContext: `context` is undefined. Seems you forgot to wrap all slider components within <Slider />"
}),
    SliderProvider = _createContext[0],
    useSliderContext = _createContext[1];

/**
 * The Slider is used to allow users to make selections from a range of values.
 * It provides context and functionality for all slider components
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices/#slider
 */
var Slider = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Slider", props);
  var ownProps = omitThemingProps(props);

  var _useTheme = useTheme(),
      direction = _useTheme.direction;

  ownProps.direction = direction;

  var _useSlider = useSlider(ownProps),
      getInputProps = _useSlider.getInputProps,
      getRootProps = _useSlider.getRootProps,
      context = _objectWithoutPropertiesLoose(_useSlider, _excluded);

  var rootProps = getRootProps();
  var inputProps = getInputProps({}, ref);
  return /*#__PURE__*/React.createElement(SliderProvider, {
    value: context
  }, /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({}, rootProps, {
    className: "chakra-slider",
    __css: styles.container
  }), props.children, /*#__PURE__*/React.createElement("input", inputProps))));
});
Slider.defaultProps = {
  orientation: "horizontal"
};

if (__DEV__) {
  Slider.displayName = "Slider";
}

/**
 * Slider component that acts as the handle used to select predefined
 * values by dragging its handle along the track
 */
var SliderThumb = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useSliderContext = useSliderContext(),
      getThumbProps = _useSliderContext.getThumbProps;

  var styles = useStyles();
  var thumbProps = getThumbProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, thumbProps, {
    className: cx("chakra-slider__thumb", props.className),
    __css: styles.thumb
  }));
});

if (__DEV__) {
  SliderThumb.displayName = "SliderThumb";
}

var SliderTrack = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useSliderContext2 = useSliderContext(),
      getTrackProps = _useSliderContext2.getTrackProps;

  var styles = useStyles();
  var trackProps = getTrackProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, trackProps, {
    className: cx("chakra-slider__track", props.className),
    __css: styles.track
  }));
});

if (__DEV__) {
  SliderTrack.displayName = "SliderTrack";
}

var SliderFilledTrack = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useSliderContext3 = useSliderContext(),
      getInnerTrackProps = _useSliderContext3.getInnerTrackProps;

  var styles = useStyles();
  var trackProps = getInnerTrackProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, trackProps, {
    className: "chakra-slider__filled-track",
    __css: styles.filledTrack
  }));
});

if (__DEV__) {
  SliderFilledTrack.displayName = "SliderFilledTrack";
}

/**
 * SliderMark is used to provide names for specific Slider
 * values by defining labels or markers along the track.
 *
 * @see Docs https://chakra-ui.com/slider
 */
var SliderMark = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useSliderContext4 = useSliderContext(),
      getMarkerProps = _useSliderContext4.getMarkerProps;

  var markProps = getMarkerProps(props, ref);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, markProps, {
    className: cx("chakra-slider__marker", props.className)
  }));
});

if (__DEV__) {
  SliderMark.displayName = "SliderMark";
}

export { RangeSlider, RangeSliderFilledTrack, RangeSliderMark, RangeSliderProvider, RangeSliderThumb, RangeSliderTrack, Slider, SliderFilledTrack, SliderMark, SliderProvider, SliderThumb, SliderTrack, useRangeSlider, useRangeSliderContext, useSlider, useSliderContext };
