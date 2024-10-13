'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var formControl = require('@chakra-ui/form-control');
var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
var React = require('react');
var icon = require('@chakra-ui/icon');
var counter = require('@chakra-ui/counter');
var hooks = require('@chakra-ui/hooks');

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

var TriangleDownIcon = function TriangleDownIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    viewBox: "0 0 24 24"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M21,5H3C2.621,5,2.275,5.214,2.105,5.553C1.937,5.892,1.973,6.297,2.2,6.6l9,12 c0.188,0.252,0.485,0.4,0.8,0.4s0.611-0.148,0.8-0.4l9-12c0.228-0.303,0.264-0.708,0.095-1.047C21.725,5.214,21.379,5,21,5z"
  }));
};
var TriangleUpIcon = function TriangleUpIcon(props) {
  return /*#__PURE__*/React__namespace.createElement(icon.Icon, _extends({
    viewBox: "0 0 24 24"
  }, props), /*#__PURE__*/React__namespace.createElement("path", {
    fill: "currentColor",
    d: "M12.8,5.4c-0.377-0.504-1.223-0.504-1.6,0l-9,12c-0.228,0.303-0.264,0.708-0.095,1.047 C2.275,18.786,2.621,19,3,19h18c0.379,0,0.725-0.214,0.895-0.553c0.169-0.339,0.133-0.744-0.095-1.047L12.8,5.4z"
  }));
};

/**
 * When click and hold on a button - the speed of auto changing the value.
 */

var CONTINUOUS_CHANGE_INTERVAL = 50;
/**
 * When click and hold on a button - the delay before auto changing the value.
 */

var CONTINUOUS_CHANGE_DELAY = 300;

/**
 * React hook used in the number input to spin its
 * value on long press of the spin buttons
 *
 * @param increment the function to increment
 * @param decrement the function to decrement
 */
function useSpinner(increment, decrement) {
  /**
   * To keep incrementing/decrementing on press, we call that `spinning`
   */
  var _useState = React.useState(false),
      isSpinning = _useState[0],
      setIsSpinning = _useState[1]; // This state keeps track of the action ("increment" or "decrement")


  var _useState2 = React.useState(null),
      action = _useState2[0],
      setAction = _useState2[1]; // To increment the value the first time you mousedown, we call that `runOnce`


  var _useState3 = React.useState(true),
      runOnce = _useState3[0],
      setRunOnce = _useState3[1]; // Store the timeout instance id in a ref, so we can clear the timeout later


  var timeoutRef = React.useRef(null); // Clears the timeout from memory

  var removeTimeout = function removeTimeout() {
    return clearTimeout(timeoutRef.current);
  };
  /**
   * useInterval hook provides a performant way to
   * update the state value at specific interval
   */


  hooks.useInterval(function () {
    if (action === "increment") {
      increment();
    }

    if (action === "decrement") {
      decrement();
    }
  }, isSpinning ? CONTINUOUS_CHANGE_INTERVAL : null); // Function to activate the spinning and increment the value

  var up = React.useCallback(function () {
    // increment the first fime
    if (runOnce) {
      increment();
    } // after a delay, keep incrementing at interval ("spinning up")


    timeoutRef.current = setTimeout(function () {
      setRunOnce(false);
      setIsSpinning(true);
      setAction("increment");
    }, CONTINUOUS_CHANGE_DELAY);
  }, [increment, runOnce]); // Function to activate the spinning and increment the value

  var down = React.useCallback(function () {
    // decrement the first fime
    if (runOnce) {
      decrement();
    } // after a delay, keep decrementing at interval ("spinning down")


    timeoutRef.current = setTimeout(function () {
      setRunOnce(false);
      setIsSpinning(true);
      setAction("decrement");
    }, CONTINUOUS_CHANGE_DELAY);
  }, [decrement, runOnce]); // Function to stop spinng (useful for mouseup, keyup handlers)

  var stop = React.useCallback(function () {
    setRunOnce(true);
    setIsSpinning(false);
    removeTimeout();
  }, []);
  /**
   * If the component unmounts while spinning,
   * let's clear the timeout as well
   */

  hooks.useUnmountEffect(removeTimeout);
  return {
    up: up,
    down: down,
    stop: stop
  };
}

var FLOATING_POINT_REGEX = /^[Ee0-9+\-.]$/;
/**
 * Determine if a character is a DOM floating point character
 * @see https://www.w3.org/TR/2012/WD-html-markup-20120329/datatypes.html#common.data.float
 */

function isFloatingPointNumericCharacter(character) {
  return FLOATING_POINT_REGEX.test(character);
}
/**
 * Determine if the event is a valid numeric keyboard event.
 * We use this so we can prevent non-number characters in the input
 */

function isValidNumericKeyboardEvent(event) {
  if (event.key == null) return true;
  var isModifierKey = event.ctrlKey || event.altKey || event.metaKey;

  if (isModifierKey) {
    return true;
  }

  var isSingleCharacterKey = event.key.length === 1;

  if (!isSingleCharacterKey) {
    return true;
  }

  return isFloatingPointNumericCharacter(event.key);
}

var _excluded$1 = ["focusInputOnChange", "clampValueOnBlur", "keepWithinRange", "min", "max", "step", "isReadOnly", "isDisabled", "isRequired", "getAriaValueText", "isInvalid", "pattern", "inputMode", "allowMouseWheel", "id", "onChange", "precision", "name", "aria-describedby", "aria-label", "aria-labelledby", "onFocus", "onBlur"];

var sanitize = function sanitize(value) {
  return value.split("").filter(isFloatingPointNumericCharacter).join("");
};
/**
 * React hook that implements the WAI-ARIA Spin Button widget
 * and used to create numeric input fields.
 *
 * It returns prop getters you can use to build your own
 * custom number inputs.
 *
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices-1.1/#spinbutton
 * @see Docs     https://www.chakra-ui.com/useNumberInput
 * @see WHATWG   https://html.spec.whatwg.org/multipage/input.html#number-state-(type=number)
 */


function useNumberInput(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      _props$focusInputOnCh = _props.focusInputOnChange,
      focusInputOnChange = _props$focusInputOnCh === void 0 ? true : _props$focusInputOnCh,
      _props$clampValueOnBl = _props.clampValueOnBlur,
      clampValueOnBlur = _props$clampValueOnBl === void 0 ? true : _props$clampValueOnBl,
      _props$keepWithinRang = _props.keepWithinRange,
      keepWithinRange = _props$keepWithinRang === void 0 ? true : _props$keepWithinRang,
      _props$min = _props.min,
      min = _props$min === void 0 ? utils.minSafeInteger : _props$min,
      _props$max = _props.max,
      max = _props$max === void 0 ? utils.maxSafeInteger : _props$max,
      _props$step = _props.step,
      stepProp = _props$step === void 0 ? 1 : _props$step,
      isReadOnly = _props.isReadOnly,
      isDisabled = _props.isDisabled,
      isRequired = _props.isRequired,
      getAriaValueText = _props.getAriaValueText,
      isInvalid = _props.isInvalid,
      _props$pattern = _props.pattern,
      pattern = _props$pattern === void 0 ? "[0-9]*(.[0-9]+)?" : _props$pattern,
      _props$inputMode = _props.inputMode,
      inputMode = _props$inputMode === void 0 ? "decimal" : _props$inputMode,
      allowMouseWheel = _props.allowMouseWheel,
      id = _props.id;
      _props.onChange;
      _props.precision;
      var name = _props.name,
      ariaDescBy = _props["aria-describedby"],
      ariaLabel = _props["aria-label"],
      ariaLabelledBy = _props["aria-labelledby"],
      onFocus = _props.onFocus,
      onBlur = _props.onBlur,
      htmlProps = _objectWithoutPropertiesLoose(_props, _excluded$1);

  var onFocusProp = hooks.useCallbackRef(onFocus);
  var onBlurProp = hooks.useCallbackRef(onBlur);
  var getAriaValueTextProp = hooks.useCallbackRef(getAriaValueText);
  /**
   * Leverage the `useCounter` hook since it provides
   * the functionality to `increment`, `decrement` and `update`
   * counter values
   */

  var counter$1 = counter.useCounter(props);
  var updateFn = counter$1.update,
      incrementFn = counter$1.increment,
      decrementFn = counter$1.decrement;
  /**
   * Keep track of the focused state of the input,
   * so user can this to change the styles of the
   * `spinners`, maybe :)
   */

  var _useBoolean = hooks.useBoolean(),
      isFocused = _useBoolean[0],
      setFocused = _useBoolean[1];

  var inputRef = React__namespace.useRef(null);
  /**
   * Sync state with uncontrolled form libraries like `react-hook-form`.
   */

  hooks.useSafeLayoutEffect(function () {
    if (!inputRef.current) return;
    var notInSync = inputRef.current.value != counter$1.value;

    if (notInSync) {
      counter$1.setValue(sanitize(inputRef.current.value));
    }
  }, []);
  var isInteractive = !(isReadOnly || isDisabled);
  var increment = React__namespace.useCallback(function (step) {
    if (step === void 0) {
      step = stepProp;
    }

    if (isInteractive) {
      incrementFn(step);
    }
  }, [incrementFn, isInteractive, stepProp]);
  var decrement = React__namespace.useCallback(function (step) {
    if (step === void 0) {
      step = stepProp;
    }

    if (isInteractive) {
      decrementFn(step);
    }
  }, [decrementFn, isInteractive, stepProp]);
  /**
   * Leverage the `useSpinner` hook to spin the input's value
   * when long press on the up and down buttons.
   *
   * This leverages `setInterval` internally
   */

  var spinner = useSpinner(increment, decrement);
  /**
   * The `onChange` handler filters out any character typed
   * that isn't floating point compatible.
   */

  var onChange = React__namespace.useCallback(function (event) {
    updateFn(sanitize(event.target.value));
  }, [updateFn]);
  var onKeyDown = React__namespace.useCallback(function (event) {
    /**
     * only allow valid numeric keys
     */
    if (!isValidNumericKeyboardEvent(event)) {
      event.preventDefault();
    }
    /**
     * Keyboard Accessibility
     *
     * We want to increase or decrease the input's value
     * based on if the user the arrow keys.
     *
     * @see https://www.w3.org/TR/wai-aria-practices-1.1/#keyboard-interaction-17
     */


    var stepFactor = getStepFactor(event) * stepProp;
    var eventKey = utils.normalizeEventKey(event);
    var keyMap = {
      ArrowUp: function ArrowUp() {
        return increment(stepFactor);
      },
      ArrowDown: function ArrowDown() {
        return decrement(stepFactor);
      },
      Home: function Home() {
        return updateFn(min);
      },
      End: function End() {
        return updateFn(max);
      }
    };
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      action(event);
    }
  }, [updateFn, decrement, increment, max, min, stepProp]);

  var getStepFactor = function getStepFactor(event) {
    var ratio = 1;

    if (event.metaKey || event.ctrlKey) {
      ratio = 0.1;
    }

    if (event.shiftKey) {
      ratio = 10;
    }

    return ratio;
  };
  /**
   * If user would like to use a human-readable representation
   * of the value, rather than the value itself they can pass `getAriaValueText`
   *
   * @see https://www.w3.org/TR/wai-aria-practices-1.1/#wai-aria-roles-states-and-properties-18
   * @see https://www.w3.org/TR/wai-aria-1.1/#aria-valuetext
   */


  var ariaValueText = React__namespace.useMemo(function () {
    var text = getAriaValueTextProp == null ? void 0 : getAriaValueTextProp(counter$1.value);

    if (!utils.isNull(text)) {
      return text;
    }

    var defaultText = counter$1.value.toString(); // empty string is an invalid ARIA attribute value

    return !defaultText ? undefined : defaultText;
  }, [counter$1.value, getAriaValueTextProp]);
  /**
   * Function that clamps the input's value on blur
   */

  var validateAndClamp = React__namespace.useCallback(function () {
    var next = counter$1.value;
    if (next === "") return;

    if (counter$1.valueAsNumber < min) {
      next = min;
    }

    if (counter$1.valueAsNumber > max) {
      next = max;
    }
    /**
     * `counter.cast` does 2 things:
     *
     * - sanitize the value by using parseFloat and some Regex
     * - used to round value to computed precision or decimal points
     */


    counter$1.cast(next);
  }, [counter$1, max, min]);
  var onInputBlur = React__namespace.useCallback(function () {
    setFocused.off();

    if (clampValueOnBlur) {
      validateAndClamp();
    }
  }, [clampValueOnBlur, setFocused, validateAndClamp]);
  var focusInput = React__namespace.useCallback(function () {
    if (focusInputOnChange) {
      utils.focus(inputRef.current, {
        nextTick: true
      });
    }
  }, [focusInputOnChange]);
  var spinUp = React__namespace.useCallback(function (event) {
    event.preventDefault();
    spinner.up();
    focusInput();
  }, [focusInput, spinner]);
  var spinDown = React__namespace.useCallback(function (event) {
    event.preventDefault();
    spinner.down();
    focusInput();
  }, [focusInput, spinner]);
  var pointerDown = utils.isBrowser && !!document.documentElement.ontouchstart ? "onTouchStart" : "onMouseDown";
  hooks.useEventListener("wheel", function (event) {
    var isInputFocused = document.activeElement === inputRef.current;
    if (!allowMouseWheel || !isInputFocused) return;
    event.preventDefault();
    var stepFactor = getStepFactor(event) * stepProp;
    var direction = Math.sign(event.deltaY);

    if (direction === -1) {
      increment(stepFactor);
    } else if (direction === 1) {
      decrement(stepFactor);
    }
  }, inputRef.current, {
    passive: false
  });
  var getIncrementButtonProps = React__namespace.useCallback(function (props, ref) {
    var _extends2;

    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var disabled = isDisabled || keepWithinRange && counter$1.isAtMax;
    return _extends({}, props, (_extends2 = {
      ref: ref,
      role: "button",
      tabIndex: -1
    }, _extends2[pointerDown] = utils.callAllHandlers(props[pointerDown], spinUp), _extends2.onMouseUp = utils.callAllHandlers(props.onMouseUp, spinner.stop), _extends2.onMouseLeave = utils.callAllHandlers(props.onMouseUp, spinner.stop), _extends2.onTouchEnd = utils.callAllHandlers(props.onTouchEnd, spinner.stop), _extends2.disabled = disabled, _extends2["aria-disabled"] = utils.ariaAttr(disabled), _extends2));
  }, [pointerDown, counter$1.isAtMax, keepWithinRange, spinUp, spinner.stop, isDisabled]);
  var getDecrementButtonProps = React__namespace.useCallback(function (props, ref) {
    var _extends3;

    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var disabled = isDisabled || keepWithinRange && counter$1.isAtMin;
    return _extends({}, props, (_extends3 = {
      ref: ref,
      role: "button",
      tabIndex: -1
    }, _extends3[pointerDown] = utils.callAllHandlers(props[pointerDown], spinDown), _extends3.onMouseLeave = utils.callAllHandlers(props.onMouseLeave, spinner.stop), _extends3.onMouseUp = utils.callAllHandlers(props.onMouseUp, spinner.stop), _extends3.onTouchEnd = utils.callAllHandlers(props.onTouchEnd, spinner.stop), _extends3.disabled = disabled, _extends3["aria-disabled"] = utils.ariaAttr(disabled), _extends3));
  }, [pointerDown, counter$1.isAtMin, keepWithinRange, spinDown, spinner.stop, isDisabled]);
  var getInputProps = React__namespace.useCallback(function (props, ref) {
    var _props$readOnly, _props$readOnly2, _props$required, _props$required2;

    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({
      name: name,
      inputMode: inputMode,
      type: "text",
      pattern: pattern,
      "aria-labelledby": ariaLabelledBy,
      "aria-label": ariaLabel,
      "aria-describedby": ariaDescBy,
      id: id,
      disabled: isDisabled
    }, props, {
      readOnly: (_props$readOnly = props.readOnly) != null ? _props$readOnly : isReadOnly,
      "aria-readonly": (_props$readOnly2 = props.readOnly) != null ? _props$readOnly2 : isReadOnly,
      "aria-required": (_props$required = props.required) != null ? _props$required : isRequired,
      required: (_props$required2 = props.required) != null ? _props$required2 : isRequired,
      ref: reactUtils.mergeRefs(inputRef, ref),
      value: counter$1.value,
      role: "spinbutton",
      "aria-valuemin": min,
      "aria-valuemax": max,
      "aria-valuenow": Number.isNaN(counter$1.valueAsNumber) ? undefined : counter$1.valueAsNumber,
      "aria-invalid": utils.ariaAttr(isInvalid != null ? isInvalid : counter$1.isOutOfRange),
      "aria-valuetext": ariaValueText,
      autoComplete: "off",
      autoCorrect: "off",
      onChange: utils.callAllHandlers(props.onChange, onChange),
      onKeyDown: utils.callAllHandlers(props.onKeyDown, onKeyDown),
      onFocus: utils.callAllHandlers(props.onFocus, onFocusProp, setFocused.on),
      onBlur: utils.callAllHandlers(props.onBlur, onBlurProp, onInputBlur)
    });
  }, [name, inputMode, pattern, ariaLabelledBy, ariaLabel, ariaDescBy, id, isDisabled, isRequired, isReadOnly, isInvalid, counter$1.value, counter$1.valueAsNumber, counter$1.isOutOfRange, min, max, ariaValueText, onChange, onKeyDown, onFocusProp, setFocused.on, onBlurProp, onInputBlur]);
  return {
    value: counter$1.value,
    valueAsNumber: counter$1.valueAsNumber,
    isFocused: isFocused,
    isDisabled: isDisabled,
    isReadOnly: isReadOnly,
    getIncrementButtonProps: getIncrementButtonProps,
    getDecrementButtonProps: getDecrementButtonProps,
    getInputProps: getInputProps,
    htmlProps: htmlProps
  };
}

var _excluded = ["htmlProps"];

/**
 * React context used to communicate between components
 */
var _createContext = reactUtils.createContext({
  name: "NumberInputContext",
  errorMessage: "useNumberInputContext: `context` is undefined. Seems you forgot to wrap number-input's components within <NumberInput />"
}),
    NumberInputProvider = _createContext[0],
    useNumberInputContext = _createContext[1];

/**
 * NumberInput
 *
 * React component that provides context and logic to all
 * number input sub-components.
 *
 * It renders a `div` by default.
 *
 * @see Docs http://chakra-ui.com/numberinput
 */
var NumberInput = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("NumberInput", props);
  var ownProps = system.omitThemingProps(props);
  var controlProps = formControl.useFormControlProps(ownProps);

  var _useNumberInput = useNumberInput(controlProps),
      htmlProps = _useNumberInput.htmlProps,
      context = _objectWithoutPropertiesLoose(_useNumberInput, _excluded);

  var ctx = React__namespace.useMemo(function () {
    return context;
  }, [context]);
  return /*#__PURE__*/React__namespace.createElement(NumberInputProvider, {
    value: ctx
  }, /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, htmlProps, {
    ref: ref,
    className: utils.cx("chakra-numberinput", props.className),
    __css: _extends({
      position: "relative",
      zIndex: 0
    }, styles.root)
  }))));
});

if (utils.__DEV__) {
  NumberInput.displayName = "NumberInput";
}

/**
 * NumberInputStepper
 *
 * React component used to group the increment and decrement
 * button spinners.
 *
 * It renders a `div` by default.
 *
 * @see Docs http://chakra-ui.com/components/number-input
 */
var NumberInputStepper = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    "aria-hidden": true,
    ref: ref
  }, props, {
    __css: _extends({
      display: "flex",
      flexDirection: "column",
      position: "absolute",
      top: "0",
      insetEnd: "0px",
      margin: "1px",
      height: "calc(100% - 2px)",
      zIndex: 1
    }, styles.stepperGroup)
  }));
});

if (utils.__DEV__) {
  NumberInputStepper.displayName = "NumberInputStepper";
}

/**
 * NumberInputField
 *
 * React component that represents the actual `input` field
 * where users can type to edit numeric values.
 *
 * It renders an `input` by default and ensures only numeric
 * values can be typed.
 *
 * @see Docs http://chakra-ui.com/numberinput
 */
var NumberInputField = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _useNumberInputContex = useNumberInputContext(),
      getInputProps = _useNumberInputContex.getInputProps;

  var input = getInputProps(props, ref);
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.input, _extends({}, input, {
    className: utils.cx("chakra-numberinput__field", props.className),
    __css: _extends({
      width: "100%"
    }, styles.field)
  }));
});

if (utils.__DEV__) {
  NumberInputField.displayName = "NumberInputField";
}

var StyledStepper = system.chakra("div", {
  baseStyle: {
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    flex: 1,
    transitionProperty: "common",
    transitionDuration: "normal",
    userSelect: "none",
    cursor: "pointer",
    lineHeight: "normal"
  }
});

/**
 * NumberDecrementStepper
 *
 * React component used to decrement the number input's value
 *
 * It renders a `div` with `role=button` by default
 */
var NumberDecrementStepper = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _props$children;

  var styles = system.useStyles();

  var _useNumberInputContex2 = useNumberInputContext(),
      getDecrementButtonProps = _useNumberInputContex2.getDecrementButtonProps;

  var decrement = getDecrementButtonProps(props, ref);
  return /*#__PURE__*/React__namespace.createElement(StyledStepper, _extends({}, decrement, {
    __css: styles.stepper
  }), (_props$children = props.children) != null ? _props$children : /*#__PURE__*/React__namespace.createElement(TriangleDownIcon, null));
});

if (utils.__DEV__) {
  NumberDecrementStepper.displayName = "NumberDecrementStepper";
}

/**
 * NumberIncrementStepper
 *
 * React component used to increment the number input's value
 *
 * It renders a `div` with `role=button` by default
 */
var NumberIncrementStepper = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _props$children2;

  var _useNumberInputContex3 = useNumberInputContext(),
      getIncrementButtonProps = _useNumberInputContex3.getIncrementButtonProps;

  var increment = getIncrementButtonProps(props, ref);
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(StyledStepper, _extends({}, increment, {
    __css: styles.stepper
  }), (_props$children2 = props.children) != null ? _props$children2 : /*#__PURE__*/React__namespace.createElement(TriangleUpIcon, null));
});

if (utils.__DEV__) {
  NumberIncrementStepper.displayName = "NumberIncrementStepper";
}

exports.NumberDecrementStepper = NumberDecrementStepper;
exports.NumberIncrementStepper = NumberIncrementStepper;
exports.NumberInput = NumberInput;
exports.NumberInputField = NumberInputField;
exports.NumberInputStepper = NumberInputStepper;
exports.StyledStepper = StyledStepper;
exports.useNumberInput = useNumberInput;
