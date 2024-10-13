import { Alert, AlertIcon, AlertTitle, AlertDescription } from '@chakra-ui/alert';
import { CloseButton } from '@chakra-ui/close-button';
import { useChakra, ThemeProvider, ColorModeContext, chakra } from '@chakra-ui/system';
import defaultTheme from '@chakra-ui/theme';
import { isFunction, __DEV__, objectKeys, isBrowser, noop } from '@chakra-ui/utils';
import * as React from 'react';
import { render } from 'react-dom';
import { useIsPresent, motion, AnimatePresence } from 'framer-motion';
import { useUpdateEffect, useTimeout } from '@chakra-ui/hooks';
import ReachAlert from '@reach/alert';

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

function _setPrototypeOf(o, p) {
  _setPrototypeOf = Object.setPrototypeOf || function _setPrototypeOf(o, p) {
    o.__proto__ = p;
    return o;
  };

  return _setPrototypeOf(o, p);
}

function _inheritsLoose(subClass, superClass) {
  subClass.prototype = Object.create(superClass.prototype);
  subClass.prototype.constructor = subClass;
  _setPrototypeOf(subClass, superClass);
}

/**
 * Given an array of toasts for a specific position.
 * It returns the toast that matches the `id` passed
 */
/**
 * Given the toast manager state, finds the toast that matches
 * the id and return its position and index
 */

function findToast(toasts, id) {
  var position = getToastPosition(toasts, id);
  var index = position ? toasts[position].findIndex(function (toast) {
    return toast.id === id;
  }) : -1;
  return {
    position: position,
    index: index
  };
}
/**
 * Given the toast manager state, finds the position of the toast that
 * matches the `id`
 */

var getToastPosition = function getToastPosition(toasts, id) {
  var _Object$values$flat$f;

  return (_Object$values$flat$f = Object.values(toasts).flat().find(function (toast) {
    return toast.id === id;
  })) == null ? void 0 : _Object$values$flat$f.position;
};
/**
 * Get's the styles to be applied to a toast's container
 * based on its position in the manager
 */

function getToastStyle(position) {
  var isRighty = position.includes("right");
  var isLefty = position.includes("left");
  var alignItems = "center";
  if (isRighty) alignItems = "flex-end";
  if (isLefty) alignItems = "flex-start";
  return {
    display: "flex",
    flexDirection: "column",
    alignItems: alignItems
  };
}

/**
 * @todo After Gerrit refactors this implementation,
 * allow users to change the toast transition direction from
 * a `ToastProvider` component.
 *
 * Here's an API example:
 *
 * ```jsx
 * <ToastProvider
 *   motion={customVariants}
 *   component={CustomToastComponent}
 *   autoCloseTimeout={3000}
 *   toastSpacing={32} // this will control the `margin` value applied
 * >
 * </ToastProvider>
 * ```
 */

var toastMotionVariants = {
  initial: function initial(props) {
    var _ref;

    var position = props.position;
    var dir = ["top", "bottom"].includes(position) ? "y" : "x";
    var factor = ["top-right", "bottom-right"].includes(position) ? 1 : -1;
    if (position === "bottom") factor = 1;
    return _ref = {
      opacity: 0
    }, _ref[dir] = factor * 24, _ref;
  },
  animate: {
    opacity: 1,
    y: 0,
    x: 0,
    scale: 1,
    transition: {
      duration: 0.4,
      ease: [0.4, 0, 0.2, 1]
    }
  },
  exit: {
    opacity: 0,
    scale: 0.85,
    transition: {
      duration: 0.2,
      ease: [0.4, 0, 1, 1]
    }
  }
};
var Toast$1 = function Toast(props) {
  var id = props.id,
      message = props.message,
      onCloseComplete = props.onCloseComplete,
      onRequestRemove = props.onRequestRemove,
      _props$requestClose = props.requestClose,
      requestClose = _props$requestClose === void 0 ? false : _props$requestClose,
      _props$position = props.position,
      position = _props$position === void 0 ? "bottom" : _props$position,
      _props$duration = props.duration,
      duration = _props$duration === void 0 ? 5000 : _props$duration,
      _props$containerStyle = props.containerStyle,
      containerStyle = _props$containerStyle === void 0 ? {} : _props$containerStyle;

  var _React$useState = React.useState(duration),
      delay = _React$useState[0],
      setDelay = _React$useState[1];

  var isPresent = useIsPresent();
  useUpdateEffect(function () {
    if (!isPresent) {
      onCloseComplete == null ? void 0 : onCloseComplete();
    }
  }, [isPresent]);
  useUpdateEffect(function () {
    setDelay(duration);
  }, [duration]);

  var onMouseEnter = function onMouseEnter() {
    return setDelay(null);
  };

  var onMouseLeave = function onMouseLeave() {
    return setDelay(duration);
  };

  var close = function close() {
    if (isPresent) onRequestRemove();
  };

  React.useEffect(function () {
    if (isPresent && requestClose) {
      onRequestRemove();
    }
  }, [isPresent, requestClose, onRequestRemove]);
  useTimeout(close, delay);
  var style = React.useMemo(function () {
    return getToastStyle(position);
  }, [position]);
  return /*#__PURE__*/React.createElement(motion.li, {
    layout: true,
    className: "chakra-toast",
    variants: toastMotionVariants,
    initial: "initial",
    animate: "animate",
    exit: "exit",
    onHoverStart: onMouseEnter,
    onHoverEnd: onMouseLeave,
    custom: {
      position: position
    },
    style: style
  }, /*#__PURE__*/React.createElement(ReachAlert, {
    className: "chakra-toast__inner",
    style: _extends({
      pointerEvents: "auto",
      maxWidth: 560,
      minWidth: 300,
      margin: "0.5rem"
    }, containerStyle)
  }, isFunction(message) ? message({
    id: id,
    onClose: close
  }) : message));
};

if (__DEV__) {
  Toast$1.displayName = "Toast";
}

/**
 * Manages the creation, and removal of toasts
 * across all corners ("top", "bottom", etc.)
 */
var ToastManager = /*#__PURE__*/function (_React$Component) {
  _inheritsLoose(ToastManager, _React$Component);

  /**
   * Static id counter to create unique ids
   * for each toast
   */

  /**
   * State to track all the toast across all positions
   */
  function ToastManager(props) {
    var _this;

    _this = _React$Component.call(this, props) || this;
    _this.state = {
      top: [],
      "top-left": [],
      "top-right": [],
      "bottom-left": [],
      bottom: [],
      "bottom-right": []
    };

    _this.notify = function (message, options) {
      var toast = _this.createToast(message, options);

      var position = toast.position,
          id = toast.id;

      _this.setState(function (prevToasts) {
        var _extends2;

        var isTop = position.includes("top");
        /**
         * - If the toast is positioned at the top edges, the
         * recent toast stacks on top of the other toasts.
         *
         * - If the toast is positioned at the bottom edges, the recent
         * toast stacks below the other toasts.
         */

        var toasts = isTop ? [toast].concat(prevToasts[position]) : [].concat(prevToasts[position], [toast]);
        return _extends({}, prevToasts, (_extends2 = {}, _extends2[position] = toasts, _extends2));
      });

      return id;
    };

    _this.updateToast = function (id, options) {
      _this.setState(function (prevState) {
        var nextState = _extends({}, prevState);

        var _findToast = findToast(nextState, id),
            position = _findToast.position,
            index = _findToast.index;

        if (position && index !== -1) {
          nextState[position][index] = _extends({}, nextState[position][index], options);
        }

        return nextState;
      });
    };

    _this.closeAll = function (_temp) {
      var _ref = _temp === void 0 ? {} : _temp,
          positions = _ref.positions;

      // only one setState here for perf reasons
      // instead of spamming this.closeToast
      _this.setState(function (prev) {
        var allPositions = ["bottom", "bottom-right", "bottom-left", "top", "top-left", "top-right"];
        var positionsToClose = positions != null ? positions : allPositions;
        return positionsToClose.reduce(function (acc, position) {
          acc[position] = prev[position].map(function (toast) {
            return _extends({}, toast, {
              requestClose: true
            });
          });
          return acc;
        }, {});
      });
    };

    _this.createToast = function (message, options) {
      var _options$id, _options$position;

      ToastManager.counter += 1;
      var id = (_options$id = options.id) != null ? _options$id : ToastManager.counter;
      var position = (_options$position = options.position) != null ? _options$position : "top";
      return {
        id: id,
        message: message,
        position: position,
        duration: options.duration,
        onCloseComplete: options.onCloseComplete,
        onRequestRemove: function onRequestRemove() {
          return _this.removeToast(String(id), position);
        },
        status: options.status,
        requestClose: false,
        containerStyle: options.containerStyle
      };
    };

    _this.closeToast = function (id) {
      _this.setState(function (prevState) {
        var _extends3;

        var position = getToastPosition(prevState, id);
        if (!position) return prevState;
        return _extends({}, prevState, (_extends3 = {}, _extends3[position] = prevState[position].map(function (toast) {
          // id may be string or number
          // eslint-disable-next-line eqeqeq
          if (toast.id == id) {
            return _extends({}, toast, {
              requestClose: true
            });
          }

          return toast;
        }), _extends3));
      });
    };

    _this.removeToast = function (id, position) {
      _this.setState(function (prevState) {
        var _extends4;

        return _extends({}, prevState, (_extends4 = {}, _extends4[position] = prevState[position].filter(function (toast) {
          return toast.id != id;
        }), _extends4));
      });
    };

    _this.isVisible = function (id) {
      var _findToast2 = findToast(_this.state, id),
          position = _findToast2.position;

      return Boolean(position);
    };

    _this.getStyle = function (position) {
      var isTopOrBottom = position === "top" || position === "bottom";
      var margin = isTopOrBottom ? "0 auto" : undefined;
      var top = position.includes("top") ? "env(safe-area-inset-top, 0px)" : undefined;
      var bottom = position.includes("bottom") ? "env(safe-area-inset-bottom, 0px)" : undefined;
      var right = !position.includes("left") ? "env(safe-area-inset-right, 0px)" : undefined;
      var left = !position.includes("right") ? "env(safe-area-inset-left, 0px)" : undefined;
      return {
        position: "fixed",
        zIndex: 5500,
        pointerEvents: "none",
        display: "flex",
        flexDirection: "column",
        margin: margin,
        top: top,
        bottom: bottom,
        right: right,
        left: left
      };
    };

    var methods = {
      notify: _this.notify,
      closeAll: _this.closeAll,
      close: _this.closeToast,
      update: _this.updateToast,
      isActive: _this.isVisible
    };
    props.notify(methods);
    return _this;
  }
  /**
   * Function to actually create a toast and add it
   * to state at the specified position
   */


  var _proto = ToastManager.prototype;

  _proto.render = function render() {
    var _this2 = this;

    return objectKeys(this.state).map(function (position) {
      var toasts = _this2.state[position];
      return /*#__PURE__*/React.createElement("ul", {
        key: position,
        id: "chakra-toast-manager-" + position,
        style: _this2.getStyle(position)
      }, /*#__PURE__*/React.createElement(AnimatePresence, {
        initial: false
      }, toasts.map(function (toast) {
        return /*#__PURE__*/React.createElement(Toast$1, _extends({
          key: toast.id
        }, toast));
      })));
    });
  };

  return ToastManager;
}(React.Component);
ToastManager.counter = 0;

var portalId = "chakra-toast-portal";

var Toaster =
/**
 * Initialize the manager and mount it in the DOM
 * inside the portal node.
 *
 * @todo
 *
 * Update toast constructor to use `PortalManager`'s node or document.body.
 * Once done, we can remove the `zIndex` in `toast.manager.tsx`
 */
function Toaster() {
  var _this = this;

  this.createToast = void 0;
  this.removeAll = void 0;
  this.closeToast = void 0;
  this.updateToast = void 0;
  this.isToastActive = void 0;

  this.bindFunctions = function (methods) {
    _this.createToast = methods.notify;
    _this.removeAll = methods.closeAll;
    _this.closeToast = methods.close;
    _this.updateToast = methods.update;
    _this.isToastActive = methods.isActive;
  };

  this.notify = function (message, options) {
    if (options === void 0) {
      options = {};
    }

    return _this.createToast == null ? void 0 : _this.createToast(message, options);
  };

  this.close = function (id) {
    _this.closeToast == null ? void 0 : _this.closeToast(id);
  };

  this.closeAll = function (options) {
    _this.removeAll == null ? void 0 : _this.removeAll(options);
  };

  this.update = function (id, options) {
    if (options === void 0) {
      options = {};
    }

    _this.updateToast == null ? void 0 : _this.updateToast(id, options);
  };

  this.isActive = function (id) {
    return _this.isToastActive == null ? void 0 : _this.isToastActive(id);
  };

  if (!isBrowser) return;
  var portal;
  var existingPortal = document.getElementById(portalId);

  if (existingPortal) {
    portal = existingPortal;
  } else {
    var _document$body;

    var div = document.createElement("div");
    div.id = portalId;
    (_document$body = document.body) == null ? void 0 : _document$body.appendChild(div);
    portal = div;
  }

  render( /*#__PURE__*/React.createElement(ToastManager, {
    notify: this.bindFunctions
  }), portal);
};

var toast = new Toaster();

function getToastPlacement(position, dir) {
  var _logical$dir;

  if (!position) return;
  var logicals = {
    "top-start": {
      ltr: "top-left",
      rtl: "top-right"
    },
    "top-end": {
      ltr: "top-right",
      rtl: "top-left"
    },
    "bottom-start": {
      ltr: "bottom-left",
      rtl: "bottom-right"
    },
    "bottom-end": {
      ltr: "bottom-right",
      rtl: "bottom-left"
    }
  };
  var logical = logicals[position];
  return (_logical$dir = logical == null ? void 0 : logical[dir]) != null ? _logical$dir : position;
}

var Toast = function Toast(props) {
  var status = props.status,
      variant = props.variant,
      id = props.id,
      title = props.title,
      isClosable = props.isClosable,
      onClose = props.onClose,
      description = props.description;
  var alertTitleId = typeof id !== "undefined" ? "toast-" + id + "-title" : undefined;
  return /*#__PURE__*/React.createElement(Alert, {
    status: status,
    variant: variant,
    id: id,
    alignItems: "start",
    borderRadius: "md",
    boxShadow: "lg",
    paddingEnd: 8,
    textAlign: "start",
    width: "auto",
    "aria-labelledby": alertTitleId
  }, /*#__PURE__*/React.createElement(AlertIcon, null), /*#__PURE__*/React.createElement(chakra.div, {
    flex: "1",
    maxWidth: "100%"
  }, title && /*#__PURE__*/React.createElement(AlertTitle, {
    id: alertTitleId
  }, title), description && /*#__PURE__*/React.createElement(AlertDescription, {
    display: "block"
  }, description)), isClosable && /*#__PURE__*/React.createElement(CloseButton, {
    size: "sm",
    onClick: onClose,
    position: "absolute",
    insetEnd: 1,
    top: 1
  }));
};

var defaults = {
  duration: 5000,
  position: "bottom",
  variant: "solid"
};
var defaultStandaloneParam = {
  theme: defaultTheme,
  colorMode: "light",
  toggleColorMode: noop,
  setColorMode: noop,
  defaultOptions: defaults
};
/**
 * Create a toast from outside of React Components
 */

function createStandaloneToast(_temp) {
  var _ref = _temp === void 0 ? defaultStandaloneParam : _temp,
      _ref$theme = _ref.theme,
      theme = _ref$theme === void 0 ? defaultStandaloneParam.theme : _ref$theme,
      _ref$colorMode = _ref.colorMode,
      colorMode = _ref$colorMode === void 0 ? defaultStandaloneParam.colorMode : _ref$colorMode,
      _ref$toggleColorMode = _ref.toggleColorMode,
      toggleColorMode = _ref$toggleColorMode === void 0 ? defaultStandaloneParam.toggleColorMode : _ref$toggleColorMode,
      _ref$setColorMode = _ref.setColorMode,
      setColorMode = _ref$setColorMode === void 0 ? defaultStandaloneParam.setColorMode : _ref$setColorMode,
      _ref$defaultOptions = _ref.defaultOptions,
      defaultOptions = _ref$defaultOptions === void 0 ? defaultStandaloneParam.defaultOptions : _ref$defaultOptions;

  var renderWithProviders = function renderWithProviders(props, options) {
    return /*#__PURE__*/React.createElement(ThemeProvider, {
      theme: theme
    }, /*#__PURE__*/React.createElement(ColorModeContext.Provider, {
      value: {
        colorMode: colorMode,
        setColorMode: setColorMode,
        toggleColorMode: toggleColorMode
      }
    }, isFunction(options.render) ? options.render(props) : /*#__PURE__*/React.createElement(Toast, _extends({}, props, options))));
  };

  var toastImpl = function toastImpl(options) {
    var opts = _extends({}, defaultOptions, options);

    opts.position = getToastPlacement(opts.position, theme.direction);

    var Message = function Message(props) {
      return renderWithProviders(props, opts);
    };

    return toast.notify(Message, opts);
  };

  toastImpl.close = toast.close;
  toastImpl.closeAll = toast.closeAll; // toasts can only be updated if they have a valid id

  toastImpl.update = function (id, options) {
    if (!id) return;

    var opts = _extends({}, defaultOptions, options);

    opts.position = getToastPlacement(opts.position, theme.direction);
    toast.update(id, _extends({}, opts, {
      message: function message(props) {
        return renderWithProviders(props, opts);
      }
    }));
  };

  toastImpl.isActive = toast.isActive;
  return toastImpl;
}
/**
 * React hook used to create a function that can be used
 * to show toasts in an application.
 */

function useToast(options) {
  var _useChakra = useChakra(),
      theme = _useChakra.theme,
      setColorMode = _useChakra.setColorMode,
      toggleColorMode = _useChakra.toggleColorMode,
      colorMode = _useChakra.colorMode;

  return React.useMemo(function () {
    return createStandaloneToast({
      theme: theme,
      colorMode: colorMode,
      setColorMode: setColorMode,
      toggleColorMode: toggleColorMode,
      defaultOptions: options
    });
  }, [theme, setColorMode, toggleColorMode, colorMode, options]);
}

export { createStandaloneToast, defaultStandaloneParam, toast, useToast };
