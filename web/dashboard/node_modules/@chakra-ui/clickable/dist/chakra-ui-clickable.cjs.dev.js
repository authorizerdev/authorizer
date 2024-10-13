'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
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

function useEventListeners() {
  var listeners = React__namespace.useRef(new Map());
  var currentListeners = listeners.current;
  var add = React__namespace.useCallback(function (el, type, listener, options) {
    listeners.current.set(listener, {
      type: type,
      el: el,
      options: options
    });
    el.addEventListener(type, listener, options);
  }, []);
  var remove = React__namespace.useCallback(function (el, type, listener, options) {
    el.removeEventListener(type, listener, options);
    listeners.current["delete"](listener);
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

var _excluded = ["ref", "isDisabled", "isFocusable", "clickOnEnter", "clickOnSpace", "onMouseDown", "onMouseUp", "onClick", "onKeyDown", "onKeyUp", "tabIndex", "onMouseOver", "onMouseLeave"];

function isValidElement(event) {
  var element = event.target;
  var tagName = element.tagName,
      isContentEditable = element.isContentEditable;
  return tagName !== "INPUT" && tagName !== "TEXTAREA" && isContentEditable !== true;
}
/**
 * useClickable implements all the interactions of a native `button`
 * component with support for making it focusable even if it is disabled.
 *
 * It can be used with both native button elements or other elements (like `div`).
 */


function useClickable(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      htmlRef = _props.ref,
      isDisabled = _props.isDisabled,
      isFocusable = _props.isFocusable,
      _props$clickOnEnter = _props.clickOnEnter,
      clickOnEnter = _props$clickOnEnter === void 0 ? true : _props$clickOnEnter,
      _props$clickOnSpace = _props.clickOnSpace,
      clickOnSpace = _props$clickOnSpace === void 0 ? true : _props$clickOnSpace,
      onMouseDown = _props.onMouseDown,
      onMouseUp = _props.onMouseUp,
      onClick = _props.onClick,
      onKeyDown = _props.onKeyDown,
      onKeyUp = _props.onKeyUp,
      tabIndexProp = _props.tabIndex,
      onMouseOver = _props.onMouseOver,
      onMouseLeave = _props.onMouseLeave,
      htmlProps = _objectWithoutPropertiesLoose(_props, _excluded);
  /**
   * We'll use this to track if the element is a button element
   */


  var _React$useState = React__namespace.useState(true),
      isButton = _React$useState[0],
      setIsButton = _React$useState[1];
  /**
   * For custom button implementation, we'll use this to track when
   * we mouse down on the button, to enable use style its ":active" style
   */


  var _React$useState2 = React__namespace.useState(false),
      isPressed = _React$useState2[0],
      setIsPressed = _React$useState2[1];

  var listeners = useEventListeners();
  /**
   * The ref callback that fires as soon as the dom node is ready
   */

  var refCallback = function refCallback(node) {
    if (!node) return;

    if (node.tagName !== "BUTTON") {
      setIsButton(false);
    }
  };

  var tabIndex = isButton ? tabIndexProp : tabIndexProp || 0;
  var trulyDisabled = isDisabled && !isFocusable;
  var handleClick = React__namespace.useCallback(function (event) {
    if (isDisabled) {
      event.stopPropagation();
      event.preventDefault();
      return;
    }

    var self = event.currentTarget;
    self.focus();
    onClick == null ? void 0 : onClick(event);
  }, [isDisabled, onClick]);
  var onDocumentKeyUp = React__namespace.useCallback(function (e) {
    if (isPressed && isValidElement(e)) {
      e.preventDefault();
      e.stopPropagation();
      setIsPressed(false); // eslint-disable-next-line @typescript-eslint/no-unused-vars

      listeners.remove(document, "keyup", onDocumentKeyUp, false);
    }
  }, [isPressed, listeners]);
  var handleKeyDown = React__namespace.useCallback(function (event) {
    onKeyDown == null ? void 0 : onKeyDown(event);

    if (isDisabled || event.defaultPrevented || event.metaKey) {
      return;
    }

    if (!isValidElement(event.nativeEvent) || isButton) return;
    var shouldClickOnEnter = clickOnEnter && event.key === "Enter";
    var shouldClickOnSpace = clickOnSpace && event.key === " ";

    if (shouldClickOnSpace) {
      event.preventDefault();
      setIsPressed(true);
    }

    if (shouldClickOnEnter) {
      event.preventDefault();
      var self = event.currentTarget;
      self.click();
    }

    listeners.add(document, "keyup", onDocumentKeyUp, false);
  }, [isDisabled, isButton, onKeyDown, clickOnEnter, clickOnSpace, listeners, onDocumentKeyUp]);
  var handleKeyUp = React__namespace.useCallback(function (event) {
    onKeyUp == null ? void 0 : onKeyUp(event);
    if (isDisabled || event.defaultPrevented || event.metaKey) return;
    if (!isValidElement(event.nativeEvent) || isButton) return;
    var shouldClickOnSpace = clickOnSpace && event.key === " ";

    if (shouldClickOnSpace) {
      event.preventDefault();
      setIsPressed(false);
      var self = event.currentTarget;
      self.click();
    }
  }, [clickOnSpace, isButton, isDisabled, onKeyUp]);
  var onDocumentMouseUp = React__namespace.useCallback(function (event) {
    if (event.button !== 0) return;
    setIsPressed(false);
    listeners.remove(document, "mouseup", onDocumentMouseUp, false);
  }, [listeners]);
  var handleMouseDown = React__namespace.useCallback(function (event) {
    if (utils.isRightClick(event)) return;

    if (isDisabled) {
      event.stopPropagation();
      event.preventDefault();
      return;
    }

    if (!isButton) {
      setIsPressed(true);
    }

    var target = event.currentTarget;
    target.focus({
      preventScroll: true
    });
    listeners.add(document, "mouseup", onDocumentMouseUp, false);
    onMouseDown == null ? void 0 : onMouseDown(event);
  }, [isDisabled, isButton, onMouseDown, listeners, onDocumentMouseUp]);
  var handleMouseUp = React__namespace.useCallback(function (event) {
    if (utils.isRightClick(event)) return;

    if (!isButton) {
      setIsPressed(false);
    }

    onMouseUp == null ? void 0 : onMouseUp(event);
  }, [onMouseUp, isButton]);
  var handleMouseOver = React__namespace.useCallback(function (event) {
    if (isDisabled) {
      event.preventDefault();
      return;
    }

    onMouseOver == null ? void 0 : onMouseOver(event);
  }, [isDisabled, onMouseOver]);
  var handleMouseLeave = React__namespace.useCallback(function (event) {
    if (isPressed) {
      event.preventDefault();
      setIsPressed(false);
    }

    onMouseLeave == null ? void 0 : onMouseLeave(event);
  }, [isPressed, onMouseLeave]);
  var ref = reactUtils.mergeRefs(htmlRef, refCallback);

  if (isButton) {
    return _extends({}, htmlProps, {
      ref: ref,
      type: "button",
      "aria-disabled": trulyDisabled ? undefined : isDisabled,
      disabled: trulyDisabled,
      onClick: handleClick,
      onMouseDown: onMouseDown,
      onMouseUp: onMouseUp,
      onKeyUp: onKeyUp,
      onKeyDown: onKeyDown,
      onMouseOver: onMouseOver,
      onMouseLeave: onMouseLeave
    });
  }

  return _extends({}, htmlProps, {
    ref: ref,
    role: "button",
    "data-active": utils.dataAttr(isPressed),
    "aria-disabled": isDisabled ? "true" : undefined,
    tabIndex: trulyDisabled ? undefined : tabIndex,
    onClick: handleClick,
    onMouseDown: handleMouseDown,
    onMouseUp: handleMouseUp,
    onKeyUp: handleKeyUp,
    onKeyDown: handleKeyDown,
    onMouseOver: handleMouseOver,
    onMouseLeave: handleMouseLeave
  });
}

exports.useClickable = useClickable;
