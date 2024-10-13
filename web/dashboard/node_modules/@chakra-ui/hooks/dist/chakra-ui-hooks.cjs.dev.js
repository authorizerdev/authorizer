'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var React = require('react');
var utils = require('@chakra-ui/utils');
var copy = require('copy-to-clipboard');

function _interopDefault (e) { return e && e.__esModule ? e : { 'default': e }; }

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
var copy__default = /*#__PURE__*/_interopDefault(copy);

/**
 * React hook to manage boolean (on - off) states
 *
 * @param initialState the initial boolean state value
 */
function useBoolean(initialState) {
  if (initialState === void 0) {
    initialState = false;
  }

  var _useState = React.useState(initialState),
      value = _useState[0],
      setValue = _useState[1];

  var on = React.useCallback(function () {
    setValue(true);
  }, []);
  var off = React.useCallback(function () {
    setValue(false);
  }, []);
  var toggle = React.useCallback(function () {
    setValue(function (prev) {
      return !prev;
    });
  }, []);
  return [value, {
    on: on,
    off: off,
    toggle: toggle
  }];
}

/**
 * useSafeLayoutEffect enables us to safely call `useLayoutEffect` on the browser
 * (for SSR reasons)
 *
 * React currently throws a warning when using useLayoutEffect on the server.
 * To get around it, we can conditionally useEffect on the server (no-op) and
 * useLayoutEffect in the browser.
 *
 * @see https://gist.github.com/gaearon/e7d97cdf38a2907924ea12e4ebdf3c85
 */

var useSafeLayoutEffect = utils.isBrowser ? React__namespace.useLayoutEffect : React__namespace.useEffect;

/**
 * React hook to persist any value between renders,
 * but keeps it up-to-date if it changes.
 *
 * @param value the value or function to persist
 */

function useCallbackRef(fn, deps) {
  if (deps === void 0) {
    deps = [];
  }

  var ref = React__namespace.useRef(fn);
  useSafeLayoutEffect(function () {
    ref.current = fn;
  }); // eslint-disable-next-line react-hooks/exhaustive-deps

  return React__namespace.useCallback(function () {
    for (var _len = arguments.length, args = new Array(_len), _key = 0; _key < _len; _key++) {
      args[_key] = arguments[_key];
    }

    return ref.current == null ? void 0 : ref.current.apply(ref, args);
  }, deps);
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

var _excluded = ["timeout"];

/**
 * React hook to copy content to clipboard
 *
 * @param text the text or value to copy
 * @param {Number} [optionsOrTimeout=1500] optionsOrTimeout - delay (in ms) to switch back to initial state once copied.
 * @param {Object} optionsOrTimeout
 * @param {string} optionsOrTimeout.format - set the desired MIME type
 * @param {number} optionsOrTimeout.timeout - delay (in ms) to switch back to initial state once copied.
 */
function useClipboard(text, optionsOrTimeout) {
  if (optionsOrTimeout === void 0) {
    optionsOrTimeout = {};
  }

  var _useState = React.useState(false),
      hasCopied = _useState[0],
      setHasCopied = _useState[1];

  var _ref = typeof optionsOrTimeout === "number" ? {
    timeout: optionsOrTimeout
  } : optionsOrTimeout,
      _ref$timeout = _ref.timeout,
      timeout = _ref$timeout === void 0 ? 1500 : _ref$timeout,
      copyOptions = _objectWithoutPropertiesLoose(_ref, _excluded);

  var onCopy = React.useCallback(function () {
    var didCopy = copy__default["default"](text, copyOptions);
    setHasCopied(didCopy);
  }, [text, copyOptions]);
  React.useEffect(function () {
    var timeoutId = null;

    if (hasCopied) {
      timeoutId = window.setTimeout(function () {
        setHasCopied(false);
      }, timeout);
    }

    return function () {
      if (timeoutId) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [timeout, hasCopied]);
  return {
    value: text,
    onCopy: onCopy,
    hasCopied: hasCopied
  };
}

/**
 * Creates a constant value over the lifecycle of a component.
 *
 * Even if `useMemo` is provided an empty array as its final argument, it doesn't offer
 * a guarantee that it won't re-run for performance reasons later on. By using `useConstant`
 * you can ensure that initialisers don't execute twice or more.
 */

function useConst(init) {
  var ref = React.useRef(null);

  if (ref.current === null) {
    ref.current = typeof init === "function" ? init() : init;
  }

  return ref.current;
}

function useControllableProp(prop, state) {
  var isControlled = prop !== undefined;
  var value = isControlled && typeof prop !== "undefined" ? prop : state;
  return [isControlled, value];
}

/**
 * React hook for using controlling component state.
 * @param props
 */
function useControllableState(props) {
  var valueProp = props.value,
      defaultValue = props.defaultValue,
      onChange = props.onChange,
      _props$shouldUpdate = props.shouldUpdate,
      shouldUpdate = _props$shouldUpdate === void 0 ? function (prev, next) {
    return prev !== next;
  } : _props$shouldUpdate;
  var onChangeProp = useCallbackRef(onChange);
  var shouldUpdateProp = useCallbackRef(shouldUpdate);

  var _React$useState = React__namespace.useState(defaultValue),
      valueState = _React$useState[0],
      setValue = _React$useState[1];

  var isControlled = valueProp !== undefined;
  var value = isControlled ? valueProp : valueState;
  var updateValue = React__namespace.useCallback(function (next) {
    var nextValue = utils.runIfFn(next, value);

    if (!shouldUpdateProp(value, nextValue)) {
      return;
    }

    if (!isControlled) {
      setValue(nextValue);
    }

    onChangeProp(nextValue);
  }, [isControlled, onChangeProp, value, shouldUpdateProp]);
  return [value, updateValue];
}

/**
 * Reack hook to measure a component's dimensions
 *
 * @param ref ref of the component to measure
 * @param observe if `true`, resize and scroll observers will be turned on
 */

function useDimensions(ref, observe) {
  var _React$useState = React__namespace.useState(null),
      dimensions = _React$useState[0],
      setDimensions = _React$useState[1];

  var rafId = React__namespace.useRef();
  useSafeLayoutEffect(function () {
    if (!ref.current) return undefined;
    var node = ref.current;

    function measure() {
      rafId.current = requestAnimationFrame(function () {
        var boxModel = utils.getBox(node);
        setDimensions(boxModel);
      });
    }

    measure();

    if (observe) {
      window.addEventListener("resize", measure);
      window.addEventListener("scroll", measure);
    }

    return function () {
      if (observe) {
        window.removeEventListener("resize", measure);
        window.removeEventListener("scroll", measure);
      }

      if (rafId.current) {
        cancelAnimationFrame(rafId.current);
      }
    };
  }, [observe]);
  return dimensions;
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

// This implementation is heavily inspired by react-aria's implementation
var defaultIdContext = {
  prefix: Math.round(Math.random() * 10000000000),
  current: 0
};
var IdContext = /*#__PURE__*/React__namespace.createContext(defaultIdContext);
var IdProvider = /*#__PURE__*/React__namespace.memo(function (_ref) {
  var children = _ref.children;
  var currentContext = React__namespace.useContext(IdContext);
  var isRoot = currentContext === defaultIdContext;
  var context = React__namespace.useMemo(function () {
    return {
      prefix: isRoot ? 0 : ++currentContext.prefix,
      current: 0
    };
  }, [isRoot, currentContext]);
  return /*#__PURE__*/React__namespace.createElement(IdContext.Provider, {
    value: context
  }, children);
});
function useId(idProp, prefix) {
  var context = React__namespace.useContext(IdContext);
  return React__namespace.useMemo(function () {
    return idProp || [prefix, context.prefix, ++context.current].filter(Boolean).join("-");
  }, // eslint-disable-next-line react-hooks/exhaustive-deps
  [idProp, prefix]);
}
/**
 * Reack hook to generate ids for use in compound components
 *
 * @param idProp the external id passed from the user
 * @param prefixes array of prefixes to use
 *
 * @example
 *
 * ```js
 * const [buttonId, menuId] = useIds("52", "button", "menu")
 *
 * // buttonId will be `button-52`
 * // menuId will be `menu-52`
 * ```
 */

function useIds(idProp) {
  for (var _len = arguments.length, prefixes = new Array(_len > 1 ? _len - 1 : 0), _key = 1; _key < _len; _key++) {
    prefixes[_key - 1] = arguments[_key];
  }

  var id = useId(idProp);
  return React__namespace.useMemo(function () {
    return prefixes.map(function (prefix) {
      return prefix + "-" + id;
    });
  }, [id, prefixes]);
}
/**
 * Used to generate an id, and after render, check if that id is rendered so we know
 * if we can use it in places such as `aria-labelledby`.
 *
 * @param partId - The unique id for the component part
 *
 * @example
 * const { ref, id } = useOptionalPart<HTMLInputElement>(`${id}-label`)
 */

function useOptionalPart(partId) {
  var _React$useState = React__namespace.useState(null),
      id = _React$useState[0],
      setId = _React$useState[1];

  var ref = React__namespace.useCallback(function (node) {
    setId(node ? partId : null);
  }, [partId]);
  return {
    ref: ref,
    id: id,
    isRendered: Boolean(id)
  };
}

function useDisclosure(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      onCloseProp = _props.onClose,
      onOpenProp = _props.onOpen,
      isOpenProp = _props.isOpen,
      idProp = _props.id;
  var onOpenPropCallbackRef = useCallbackRef(onOpenProp);
  var onClosePropCallbackRef = useCallbackRef(onCloseProp);

  var _React$useState = React__namespace.useState(props.defaultIsOpen || false),
      isOpenState = _React$useState[0],
      setIsOpen = _React$useState[1];

  var _useControllableProp = useControllableProp(isOpenProp, isOpenState),
      isControlled = _useControllableProp[0],
      isOpen = _useControllableProp[1];

  var id = useId(idProp, "disclosure");
  var onClose = React__namespace.useCallback(function () {
    if (!isControlled) {
      setIsOpen(false);
    }

    onClosePropCallbackRef == null ? void 0 : onClosePropCallbackRef();
  }, [isControlled, onClosePropCallbackRef]);
  var onOpen = React__namespace.useCallback(function () {
    if (!isControlled) {
      setIsOpen(true);
    }

    onOpenPropCallbackRef == null ? void 0 : onOpenPropCallbackRef();
  }, [isControlled, onOpenPropCallbackRef]);
  var onToggle = React__namespace.useCallback(function () {
    var action = isOpen ? onClose : onOpen;
    action();
  }, [isOpen, onOpen, onClose]);
  return {
    isOpen: !!isOpen,
    onOpen: onOpen,
    onClose: onClose,
    onToggle: onToggle,
    isControlled: isControlled,
    getButtonProps: function getButtonProps(props) {
      if (props === void 0) {
        props = {};
      }

      return _extends({}, props, {
        "aria-expanded": "true",
        "aria-controls": id,
        onClick: utils.callAllHandlers(props.onClick, onToggle)
      });
    },
    getDisclosureProps: function getDisclosureProps(props) {
      if (props === void 0) {
        props = {};
      }

      return _extends({}, props, {
        hidden: !isOpen,
        id: id
      });
    }
  };
}

/**
 * React hook for performant `useCallbacks`
 *
 * @see https://github.com/facebook/react/issues/14099#issuecomment-440013892
 *
 * @deprecated Use `useCallbackRef` instead. `useEventCallback` will be removed
 * in a future version.
 */

function useEventCallback(callback) {
  var ref = React__namespace.useRef(callback);
  useSafeLayoutEffect(function () {
    ref.current = callback;
  });
  return React__namespace.useCallback(function (event) {
    for (var _len = arguments.length, args = new Array(_len > 1 ? _len - 1 : 0), _key = 1; _key < _len; _key++) {
      args[_key - 1] = arguments[_key];
    }

    return ref.current.apply(ref, [event].concat(args));
  }, []);
}

/**
 * React hook to manage browser event listeners
 *
 * @param event the event name
 * @param handler the event handler function to execute
 * @param doc the dom environment to execute against (defaults to `document`)
 * @param options the event listener options
 *
 * @internal
 */
function useEventListener(event, handler, env, options) {
  var listener = useCallbackRef(handler);
  React__namespace.useEffect(function () {
    var _runIfFn;

    var node = (_runIfFn = utils.runIfFn(env)) != null ? _runIfFn : document;
    node.addEventListener(event, listener, options);
    return function () {
      node.removeEventListener(event, listener, options);
    };
  }, [event, env, options, listener]);
  return function () {
    var _runIfFn2;

    var node = (_runIfFn2 = utils.runIfFn(env)) != null ? _runIfFn2 : document;
    node.removeEventListener(event, listener, options);
  };
}

function useEventListenerMap() {
  var listeners = React__namespace.useRef(new Map());
  var currentListeners = listeners.current;
  var add = React__namespace.useCallback(function (el, type, listener, options) {
    var pointerEventListener = utils.wrapPointerEventHandler(listener, type === "pointerdown");
    listeners.current.set(listener, {
      __listener: pointerEventListener,
      type: utils.getPointerEventName(type),
      el: el,
      options: options
    });
    el.addEventListener(type, pointerEventListener, options);
  }, []);
  var remove = React__namespace.useCallback(function (el, type, listener, options) {
    var _listeners$current$ge = listeners.current.get(listener),
        pointerEventListener = _listeners$current$ge.__listener;

    el.removeEventListener(type, pointerEventListener, options);
    listeners.current["delete"](pointerEventListener);
  }, []);
  React__namespace.useEffect(function () {
    return function () {
      currentListeners.forEach(function (value, key) {
        remove(value.el, value.type, key, value.options);
      });
    };
  }, [remove, currentListeners]);
  return {
    add: add,
    remove: remove
  };
}

/**
 * React effect hook that invokes only on update.
 * It doesn't invoke on mount
 */

var useUpdateEffect = function useUpdateEffect(effect, deps) {
  var mounted = React__namespace.useRef(false);
  React__namespace.useEffect(function () {
    if (mounted.current) {
      return effect();
    }

    mounted.current = true;
    return undefined; // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return mounted.current;
};

/**
 * React hook to focus an element conditionally
 *
 * @param ref the ref of the element to focus
 * @param options focus management options
 */
function useFocusEffect(ref, options) {
  var shouldFocus = options.shouldFocus,
      preventScroll = options.preventScroll;
  useUpdateEffect(function () {
    var node = ref.current;
    if (!node || !shouldFocus) return;

    if (!utils.hasFocusWithin(node)) {
      utils.focus(node, {
        preventScroll: preventScroll,
        nextTick: true
      });
    }
  }, [shouldFocus, ref, preventScroll]);
}

function preventReturnFocus(containerRef) {
  var el = containerRef.current;
  if (!el) return false;
  var activeElement = utils.getActiveElement(el);
  if (!activeElement) return false;
  if (utils.contains(el, activeElement)) return false;
  if (utils.isTabbable(activeElement)) return true;
  return false;
}
/**
 * Popover hook to manage the focus when the popover closes or hides.
 *
 * We either want to return focus back to the popover trigger or
 * let focus proceed normally if user moved to another interactive
 * element in the viewport.
 */


function useFocusOnHide(containerRef, options) {
  var shouldFocusProp = options.shouldFocus,
      visible = options.visible,
      focusRef = options.focusRef;
  var shouldFocus = shouldFocusProp && !visible;
  useUpdateEffect(function () {
    if (!shouldFocus) return;

    if (preventReturnFocus(containerRef)) {
      return;
    }

    var el = (focusRef == null ? void 0 : focusRef.current) || containerRef.current;

    if (el) {
      utils.focus(el, {
        nextTick: true
      });
    }
  }, [shouldFocus, containerRef, focusRef]);
}

/**
 * Credit goes to `framer-motion` of this useful utilities.
 * License can be found here: https://github.com/framer/motion
 */
/**
 * @internal
 */

function usePointerEvent(env, eventName, handler, options) {
  return useEventListener(utils.getPointerEventName(eventName), utils.wrapPointerEventHandler(handler, eventName === "pointerdown"), env, options);
}

/**
 * Polyfill to get `relatedTarget` working correctly consistently
 * across all browsers.
 *
 * It ensures that elements receives focus on pointer down if
 * it's not the active active element.
 *
 * @internal
 */
function useFocusOnPointerDown(props) {
  var ref = props.ref,
      elements = props.elements,
      enabled = props.enabled;
  var isSafari = utils.detectBrowser("Safari");

  var doc = function doc() {
    return utils.getOwnerDocument(ref.current);
  };

  usePointerEvent(doc, "pointerdown", function (event) {
    if (!isSafari || !enabled) return;
    var target = event.target;
    var els = elements != null ? elements : [ref];
    var isValidTarget = els.some(function (elementOrRef) {
      var el = utils.isRefObject(elementOrRef) ? elementOrRef.current : elementOrRef;
      return utils.contains(el, target);
    });

    if (!utils.isActiveElement(target) && isValidTarget) {
      event.preventDefault();
      utils.focus(target);
    }
  });
}

var defaultOptions = {
  preventScroll: true,
  shouldFocus: false
};
function useFocusOnShow(target, options) {
  if (options === void 0) {
    options = defaultOptions;
  }

  var _options = options,
      focusRef = _options.focusRef,
      preventScroll = _options.preventScroll,
      shouldFocus = _options.shouldFocus,
      visible = _options.visible;
  var element = utils.isRefObject(target) ? target.current : target;
  var autoFocus = shouldFocus && visible;
  var onFocus = React.useCallback(function () {
    if (!element || !autoFocus) return;
    if (utils.contains(element, document.activeElement)) return;

    if (focusRef != null && focusRef.current) {
      utils.focus(focusRef.current, {
        preventScroll: preventScroll,
        nextTick: true
      });
    } else {
      var tabbableEls = utils.getAllFocusable(element);

      if (tabbableEls.length > 0) {
        utils.focus(tabbableEls[0], {
          preventScroll: preventScroll,
          nextTick: true
        });
      }
    }
  }, [autoFocus, preventScroll, element, focusRef]);
  useUpdateEffect(function () {
    onFocus();
  }, [onFocus]);
  useEventListener("transitionend", onFocus, element);
}

function useUnmountEffect(fn, deps) {
  if (deps === void 0) {
    deps = [];
  }

  return React__namespace.useEffect(function () {
    return function () {
      return fn();
    };
  }, // eslint-disable-next-line react-hooks/exhaustive-deps
  deps);
}

function useForceUpdate() {
  var unloadingRef = React__namespace.useRef(false);

  var _React$useState = React__namespace.useState(0),
      count = _React$useState[0],
      setCount = _React$useState[1];

  useUnmountEffect(function () {
    unloadingRef.current = true;
  });
  return React__namespace.useCallback(function () {
    if (!unloadingRef.current) {
      setCount(count + 1);
    }
  }, [count]);
}

/**
 * React Hook that provides a declarative `setInterval`
 *
 * @param callback the callback to execute at interval
 * @param delay the `setInterval` delay (in ms)
 */

function useInterval(callback, delay) {
  var fn = useCallbackRef(callback);
  React__namespace.useEffect(function () {
    var intervalId = null;

    var tick = function tick() {
      return fn();
    };

    if (delay !== null) {
      intervalId = window.setInterval(tick, delay);
    }

    return function () {
      if (intervalId) {
        window.clearInterval(intervalId);
      }
    };
  }, [delay, fn]);
}

/**
 * React hook to persist any value between renders,
 * but keeps it up-to-date if it changes.
 *
 * @param value the value or function to persist
 */

function useLatestRef(value) {
  var ref = React__namespace.useRef(null);
  ref.current = value;
  return ref;
}

/* eslint-disable react-hooks/exhaustive-deps */
function assignRef(ref, value) {
  if (ref == null) return;

  if (typeof ref === "function") {
    ref(value);
    return;
  }

  try {
    // @ts-ignore
    ref.current = value;
  } catch (error) {
    throw new Error("Cannot assign value '" + value + "' to ref '" + ref + "'");
  }
}
/**
 * React hook that merges react refs into a single memoized function
 *
 * @example
 * import React from "react";
 * import { useMergeRefs } from `@chakra-ui/hooks`;
 *
 * const Component = React.forwardRef((props, ref) => {
 *   const internalRef = React.useRef();
 *   return <div {...props} ref={useMergeRefs(internalRef, ref)} />;
 * });
 */

function useMergeRefs() {
  for (var _len = arguments.length, refs = new Array(_len), _key = 0; _key < _len; _key++) {
    refs[_key] = arguments[_key];
  }

  return React__namespace.useMemo(function () {
    if (refs.every(function (ref) {
      return ref == null;
    })) {
      return null;
    }

    return function (node) {
      refs.forEach(function (ref) {
        if (ref) assignRef(ref, node);
      });
    };
  }, refs);
}

/**
 * @deprecated `useMouseDownRef` will be removed in a future version.
 */

function useMouseDownRef(shouldListen) {
  if (shouldListen === void 0) {
    shouldListen = true;
  }

  var mouseDownRef = React__namespace["default"].useRef();
  useEventListener("mousedown", function (event) {
    if (shouldListen) {
      mouseDownRef.current = event.target;
    }
  });
  return mouseDownRef;
}

/**
 * Example, used in components like Dialogs and Popovers so they can close
 * when a user clicks outside them.
 */
function useOutsideClick(props) {
  var ref = props.ref,
      handler = props.handler,
      _props$enabled = props.enabled,
      enabled = _props$enabled === void 0 ? true : _props$enabled;
  var savedHandler = useCallbackRef(handler);
  var stateRef = React.useRef({
    isPointerDown: false,
    ignoreEmulatedMouseEvents: false
  });
  var state = stateRef.current;
  React.useEffect(function () {
    if (!enabled) return;

    var onPointerDown = function onPointerDown(e) {
      if (isValidEvent(e, ref)) {
        state.isPointerDown = true;
      }
    };

    var onMouseUp = function onMouseUp(event) {
      if (state.ignoreEmulatedMouseEvents) {
        state.ignoreEmulatedMouseEvents = false;
        return;
      }

      if (state.isPointerDown && handler && isValidEvent(event, ref)) {
        state.isPointerDown = false;
        savedHandler(event);
      }
    };

    var onTouchEnd = function onTouchEnd(event) {
      state.ignoreEmulatedMouseEvents = true;

      if (handler && state.isPointerDown && isValidEvent(event, ref)) {
        state.isPointerDown = false;
        savedHandler(event);
      }
    };

    var doc = utils.getOwnerDocument(ref.current);
    doc.addEventListener("mousedown", onPointerDown, true);
    doc.addEventListener("mouseup", onMouseUp, true);
    doc.addEventListener("touchstart", onPointerDown, true);
    doc.addEventListener("touchend", onTouchEnd, true);
    return function () {
      doc.removeEventListener("mousedown", onPointerDown, true);
      doc.removeEventListener("mouseup", onMouseUp, true);
      doc.removeEventListener("touchstart", onPointerDown, true);
      doc.removeEventListener("touchend", onTouchEnd, true);
    };
  }, [handler, ref, savedHandler, state, enabled]);
}

function isValidEvent(event, ref) {
  var _ref$current;

  var target = event.target;
  if (event.button > 0) return false; // if the event target is no longer in the document

  if (target) {
    var doc = utils.getOwnerDocument(target);
    if (!doc.body.contains(target)) return false;
  }

  return !((_ref$current = ref.current) != null && _ref$current.contains(target));
}

function usePanGesture(ref, props) {
  var onPan = props.onPan,
      onPanStart = props.onPanStart,
      onPanEnd = props.onPanEnd,
      onPanSessionStart = props.onPanSessionStart,
      onPanSessionEnd = props.onPanSessionEnd,
      threshold = props.threshold;
  var hasPanEvents = Boolean(onPan || onPanStart || onPanEnd || onPanSessionStart || onPanSessionEnd);
  var panSession = React.useRef(null);
  var handlers = {
    onSessionStart: onPanSessionStart,
    onSessionEnd: onPanSessionEnd,
    onStart: onPanStart,
    onMove: onPan,
    onEnd: function onEnd(event, info) {
      panSession.current = null;
      onPanEnd == null ? void 0 : onPanEnd(event, info);
    }
  };
  React.useEffect(function () {
    var _panSession$current;

    (_panSession$current = panSession.current) == null ? void 0 : _panSession$current.updateHandlers(handlers);
  });

  function onPointerDown(event) {
    panSession.current = new utils.PanSession(event, handlers, threshold);
  }

  usePointerEvent(function () {
    return ref.current;
  }, "pointerdown", hasPanEvents ? onPointerDown : utils.noop);
  useUnmountEffect(function () {
    var _panSession$current2;

    (_panSession$current2 = panSession.current) == null ? void 0 : _panSession$current2.end();
    panSession.current = null;
  });
}

function usePrevious(value) {
  var ref = React.useRef();
  React.useEffect(function () {
    ref.current = value;
  }, [value]);
  return ref.current;
}

/**
 * Checks if the key pressed is a printable character
 * and can be used for shortcut navigation
 */

function isPrintableCharacter(event) {
  var key = event.key;
  return key.length === 1 || key.length > 1 && /[^a-zA-Z0-9]/.test(key);
}

/**
 * React hook that provides an enhanced keydown handler,
 * that's used for key navigation within menus, select dropdowns.
 */
function useShortcut(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      _props$timeout = _props.timeout,
      timeout = _props$timeout === void 0 ? 300 : _props$timeout,
      _props$preventDefault = _props.preventDefault,
      preventDefault = _props$preventDefault === void 0 ? function () {
    return true;
  } : _props$preventDefault;

  var _React$useState = React__namespace.useState([]),
      keys = _React$useState[0],
      setKeys = _React$useState[1];

  var timeoutRef = React__namespace.useRef();

  var flush = function flush() {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  };

  var clearKeysAfterDelay = function clearKeysAfterDelay() {
    flush();
    timeoutRef.current = setTimeout(function () {
      setKeys([]);
      timeoutRef.current = null;
    }, timeout);
  };

  React__namespace.useEffect(function () {
    return flush;
  }, []);

  function onKeyDown(fn) {
    return function (event) {
      if (event.key === "Backspace") {
        var keysCopy = [].concat(keys);
        keysCopy.pop();
        setKeys(keysCopy);
        return;
      }

      if (isPrintableCharacter(event)) {
        var _keysCopy = keys.concat(event.key);

        if (preventDefault(event)) {
          event.preventDefault();
          event.stopPropagation();
        }

        setKeys(_keysCopy);
        fn(_keysCopy.join(""));
        clearKeysAfterDelay();
      }
    };
  }

  return onKeyDown;
}

/**
 * React hook that provides a declarative `setTimeout`
 *
 * @param callback the callback to run after specified delay
 * @param delay the delay (in ms)
 */

function useTimeout(callback, delay) {
  var fn = useCallbackRef(callback);
  React__namespace.useEffect(function () {
    if (delay == null) return undefined;
    var timeoutId = null;
    timeoutId = window.setTimeout(function () {
      fn();
    }, delay);
    return function () {
      if (timeoutId) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [delay, fn]);
}

function useWhyDidYouUpdate(name, props) {
  var previousProps = React__namespace.useRef();
  React__namespace.useEffect(function () {
    if (previousProps.current) {
      var allKeys = Object.keys(_extends({}, previousProps.current, props));
      var changesObj = {};
      allKeys.forEach(function (key) {
        if (previousProps.current[key] !== props[key]) {
          changesObj[key] = {
            from: previousProps.current[key],
            to: props[key]
          };
        }
      });

      if (Object.keys(changesObj).length) {
        console.log("[why-did-you-update]", name, changesObj);
      }
    }

    previousProps.current = props;
  });
}

exports.IdProvider = IdProvider;
exports.assignRef = assignRef;
exports.useBoolean = useBoolean;
exports.useCallbackRef = useCallbackRef;
exports.useClipboard = useClipboard;
exports.useConst = useConst;
exports.useControllableProp = useControllableProp;
exports.useControllableState = useControllableState;
exports.useDimensions = useDimensions;
exports.useDisclosure = useDisclosure;
exports.useEventCallback = useEventCallback;
exports.useEventListener = useEventListener;
exports.useEventListenerMap = useEventListenerMap;
exports.useFocusEffect = useFocusEffect;
exports.useFocusOnHide = useFocusOnHide;
exports.useFocusOnPointerDown = useFocusOnPointerDown;
exports.useFocusOnShow = useFocusOnShow;
exports.useForceUpdate = useForceUpdate;
exports.useId = useId;
exports.useIds = useIds;
exports.useInterval = useInterval;
exports.useLatestRef = useLatestRef;
exports.useMergeRefs = useMergeRefs;
exports.useMouseDownRef = useMouseDownRef;
exports.useOptionalPart = useOptionalPart;
exports.useOutsideClick = useOutsideClick;
exports.usePanGesture = usePanGesture;
exports.usePointerEvent = usePointerEvent;
exports.usePrevious = usePrevious;
exports.useSafeLayoutEffect = useSafeLayoutEffect;
exports.useShortcut = useShortcut;
exports.useTimeout = useTimeout;
exports.useUnmountEffect = useUnmountEffect;
exports.useUpdateEffect = useUpdateEffect;
exports.useWhyDidYouUpdate = useWhyDidYouUpdate;
