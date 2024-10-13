import { CloseButton } from '@chakra-ui/close-button';
import { FocusLock } from '@chakra-ui/focus-lock';
import { Portal } from '@chakra-ui/portal';
import { chakra, useMultiStyleConfig, StylesProvider, forwardRef, useStyles, useTheme } from '@chakra-ui/system';
import { slideFadeConfig, scaleFadeConfig, fadeConfig, Slide } from '@chakra-ui/transition';
import { callAllHandlers, __DEV__, cx } from '@chakra-ui/utils';
import { mergeRefs, createContext } from '@chakra-ui/react-utils';
import { motion, AnimatePresence, usePresence } from 'framer-motion';
import * as React from 'react';
import { useEffect, useRef, useCallback, useState } from 'react';
import { RemoveScroll } from 'react-remove-scroll';
import { useIds } from '@chakra-ui/hooks';
import { hideOthers } from 'aria-hidden';

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

var _excluded$3 = ["preset"];
var transitions = {
  slideInBottom: _extends({}, slideFadeConfig, {
    custom: {
      offsetY: 16,
      reverse: true
    }
  }),
  slideInRight: _extends({}, slideFadeConfig, {
    custom: {
      offsetX: 16,
      reverse: true
    }
  }),
  scale: _extends({}, scaleFadeConfig, {
    custom: {
      initialScale: 0.95,
      reverse: true
    }
  }),
  none: {}
};
var Motion = chakra(motion.section);
var ModalTransition = /*#__PURE__*/React.forwardRef(function (props, ref) {
  var preset = props.preset,
      rest = _objectWithoutPropertiesLoose(props, _excluded$3);

  var motionProps = transitions[preset];
  return /*#__PURE__*/React.createElement(Motion, _extends({
    ref: ref
  }, motionProps, rest));
});

/**
 * Proper state management for nested modals.
 * Simplified, but inspired by material-ui's ModalManager class.
 */

var ModalManager = /*#__PURE__*/function () {
  function ModalManager() {
    this.modals = void 0;
    this.modals = [];
  }

  var _proto = ModalManager.prototype;

  _proto.add = function add(modal) {
    this.modals.push(modal);
  };

  _proto.remove = function remove(modal) {
    this.modals = this.modals.filter(function (_modal) {
      return _modal !== modal;
    });
  };

  _proto.isTopModal = function isTopModal(modal) {
    var topmostModal = this.modals[this.modals.length - 1];
    return topmostModal === modal;
  };

  return ModalManager;
}();

var manager = new ModalManager();
function useModalManager(ref, isOpen) {
  useEffect(function () {
    if (isOpen) {
      manager.add(ref);
    }

    return function () {
      manager.remove(ref);
    };
  }, [isOpen, ref]);
}

/**
 * Modal hook that manages all the logic for the modal dialog widget
 * and returns prop getters, state and actions.
 *
 * @param props
 */
function useModal(props) {
  var isOpen = props.isOpen,
      onClose = props.onClose,
      id = props.id,
      _props$closeOnOverlay = props.closeOnOverlayClick,
      closeOnOverlayClick = _props$closeOnOverlay === void 0 ? true : _props$closeOnOverlay,
      _props$closeOnEsc = props.closeOnEsc,
      closeOnEsc = _props$closeOnEsc === void 0 ? true : _props$closeOnEsc,
      _props$useInert = props.useInert,
      useInert = _props$useInert === void 0 ? true : _props$useInert,
      onOverlayClickProp = props.onOverlayClick,
      onEsc = props.onEsc;
  var dialogRef = useRef(null);
  var overlayRef = useRef(null);

  var _useIds = useIds(id, "chakra-modal", "chakra-modal--header", "chakra-modal--body"),
      dialogId = _useIds[0],
      headerId = _useIds[1],
      bodyId = _useIds[2];
  /**
   * Hook used to polyfill `aria-modal` for older browsers.
   * It uses `aria-hidden` to all other nodes.
   *
   * @see https://developer.paciellogroup.com/blog/2018/06/the-current-state-of-modal-dialog-accessibility/
   */


  useAriaHidden(dialogRef, isOpen && useInert);
  /**
   * Hook use to manage multiple or nested modals
   */

  useModalManager(dialogRef, isOpen);
  var mouseDownTarget = useRef(null);
  var onMouseDown = useCallback(function (event) {
    mouseDownTarget.current = event.target;
  }, []);
  var onKeyDown = useCallback(function (event) {
    if (event.key === "Escape") {
      event.stopPropagation();

      if (closeOnEsc) {
        onClose == null ? void 0 : onClose();
      }

      onEsc == null ? void 0 : onEsc();
    }
  }, [closeOnEsc, onClose, onEsc]);

  var _useState = useState(false),
      headerMounted = _useState[0],
      setHeaderMounted = _useState[1];

  var _useState2 = useState(false),
      bodyMounted = _useState2[0],
      setBodyMounted = _useState2[1];

  var getDialogProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({
      role: "dialog"
    }, props, {
      ref: mergeRefs(ref, dialogRef),
      id: dialogId,
      tabIndex: -1,
      "aria-modal": true,
      "aria-labelledby": headerMounted ? headerId : undefined,
      "aria-describedby": bodyMounted ? bodyId : undefined,
      onClick: callAllHandlers(props.onClick, function (event) {
        return event.stopPropagation();
      })
    });
  }, [bodyId, bodyMounted, dialogId, headerId, headerMounted]);
  var onOverlayClick = useCallback(function (event) {
    event.stopPropagation();
    /**
     * Make sure the event starts and ends on the same DOM element.
     *
     * This is used to prevent the modal from closing when you
     * start dragging from the content, and release drag outside the content.
     *
     * We prevent this because it is technically not a considered "click outside"
     */

    if (mouseDownTarget.current !== event.target) return;
    /**
     * When you click on the overlay, we want to remove only the topmost modal
     */

    if (!manager.isTopModal(dialogRef)) return;

    if (closeOnOverlayClick) {
      onClose == null ? void 0 : onClose();
    }

    onOverlayClickProp == null ? void 0 : onOverlayClickProp();
  }, [onClose, closeOnOverlayClick, onOverlayClickProp]);
  var getDialogContainerProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: mergeRefs(ref, overlayRef),
      onClick: callAllHandlers(props.onClick, onOverlayClick),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown),
      onMouseDown: callAllHandlers(props.onMouseDown, onMouseDown)
    });
  }, [onKeyDown, onMouseDown, onOverlayClick]);
  return {
    isOpen: isOpen,
    onClose: onClose,
    headerId: headerId,
    bodyId: bodyId,
    setBodyMounted: setBodyMounted,
    setHeaderMounted: setHeaderMounted,
    dialogRef: dialogRef,
    overlayRef: overlayRef,
    getDialogProps: getDialogProps,
    getDialogContainerProps: getDialogContainerProps
  };
}

/**
 * Modal hook to polyfill `aria-modal`.
 *
 * It applies `aria-hidden` to elements behind the modal
 * to indicate that they're `inert`.
 *
 * @param ref React ref of the node
 * @param shouldHide whether `aria-hidden` should be applied
 */
function useAriaHidden(ref, shouldHide) {
  useEffect(function () {
    if (!ref.current) return undefined;
    var undo = null;

    if (shouldHide && ref.current) {
      undo = hideOthers(ref.current);
    }

    return function () {
      if (shouldHide) {
        undo == null ? void 0 : undo();
      }
    };
  }, [shouldHide, ref]);
}

var _excluded$2 = ["className", "children", "containerProps"],
    _excluded2$1 = ["className", "transition"],
    _excluded3 = ["className"],
    _excluded4 = ["className"],
    _excluded5 = ["className"],
    _excluded6 = ["onClick", "className"];

var _createContext$1 = createContext({
  strict: true,
  name: "ModalContext",
  errorMessage: "useModalContext: `context` is undefined. Seems you forgot to wrap modal components in `<Modal />`"
}),
    ModalContextProvider = _createContext$1[0],
    useModalContext = _createContext$1[1];
/**
 * Modal provides context, theming, and accessibility properties
 * to all other modal components.
 *
 * It doesn't render any DOM node.
 */

var Modal = function Modal(props) {
  var portalProps = props.portalProps,
      children = props.children,
      autoFocus = props.autoFocus,
      trapFocus = props.trapFocus,
      initialFocusRef = props.initialFocusRef,
      finalFocusRef = props.finalFocusRef,
      returnFocusOnClose = props.returnFocusOnClose,
      blockScrollOnMount = props.blockScrollOnMount,
      allowPinchZoom = props.allowPinchZoom,
      preserveScrollBarGap = props.preserveScrollBarGap,
      motionPreset = props.motionPreset,
      lockFocusAcrossFrames = props.lockFocusAcrossFrames;
  var styles = useMultiStyleConfig("Modal", props);
  var modal = useModal(props);

  var context = _extends({}, modal, {
    autoFocus: autoFocus,
    trapFocus: trapFocus,
    initialFocusRef: initialFocusRef,
    finalFocusRef: finalFocusRef,
    returnFocusOnClose: returnFocusOnClose,
    blockScrollOnMount: blockScrollOnMount,
    allowPinchZoom: allowPinchZoom,
    preserveScrollBarGap: preserveScrollBarGap,
    motionPreset: motionPreset,
    lockFocusAcrossFrames: lockFocusAcrossFrames
  });

  return /*#__PURE__*/React.createElement(ModalContextProvider, {
    value: context
  }, /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(AnimatePresence, null, context.isOpen && /*#__PURE__*/React.createElement(Portal, portalProps, children))));
};
Modal.defaultProps = {
  lockFocusAcrossFrames: true,
  returnFocusOnClose: true,
  scrollBehavior: "outside",
  trapFocus: true,
  autoFocus: true,
  blockScrollOnMount: true,
  allowPinchZoom: false,
  motionPreset: "scale"
};

if (__DEV__) {
  Modal.displayName = "Modal";
}

var MotionDiv = chakra(motion.div);
/**
 * ModalContent is used to group modal's content. It has all the
 * necessary `aria-*` properties to indicate that it is a modal
 */

var ModalContent = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      children = props.children,
      rootProps = props.containerProps,
      rest = _objectWithoutPropertiesLoose(props, _excluded$2);

  var _useModalContext = useModalContext(),
      getDialogProps = _useModalContext.getDialogProps,
      getDialogContainerProps = _useModalContext.getDialogContainerProps;

  var dialogProps = getDialogProps(rest, ref);
  var containerProps = getDialogContainerProps(rootProps);

  var _className = cx("chakra-modal__content", className);

  var styles = useStyles();

  var dialogStyles = _extends({
    display: "flex",
    flexDirection: "column",
    position: "relative",
    width: "100%",
    outline: 0
  }, styles.dialog);

  var dialogContainerStyles = _extends({
    display: "flex",
    width: "100vw",
    height: "100vh",
    "@supports(height: -webkit-fill-available)": {
      height: "-webkit-fill-available"
    },
    position: "fixed",
    left: 0,
    top: 0
  }, styles.dialogContainer);

  var _useModalContext2 = useModalContext(),
      motionPreset = _useModalContext2.motionPreset;

  return /*#__PURE__*/React.createElement(ModalFocusScope, null, /*#__PURE__*/React.createElement(chakra.div, _extends({}, containerProps, {
    className: "chakra-modal__content-container" // tabindex="-1" means that the element is not reachable via sequential keyboard navigation, @see #4686
    ,
    tabIndex: -1,
    __css: dialogContainerStyles
  }), /*#__PURE__*/React.createElement(ModalTransition, _extends({
    preset: motionPreset,
    className: _className
  }, dialogProps, {
    __css: dialogStyles
  }), children)));
});

if (__DEV__) {
  ModalContent.displayName = "ModalContent";
}

function ModalFocusScope(props) {
  var _useModalContext3 = useModalContext(),
      autoFocus = _useModalContext3.autoFocus,
      trapFocus = _useModalContext3.trapFocus,
      dialogRef = _useModalContext3.dialogRef,
      initialFocusRef = _useModalContext3.initialFocusRef,
      blockScrollOnMount = _useModalContext3.blockScrollOnMount,
      allowPinchZoom = _useModalContext3.allowPinchZoom,
      finalFocusRef = _useModalContext3.finalFocusRef,
      returnFocusOnClose = _useModalContext3.returnFocusOnClose,
      preserveScrollBarGap = _useModalContext3.preserveScrollBarGap,
      lockFocusAcrossFrames = _useModalContext3.lockFocusAcrossFrames;

  var _usePresence = usePresence(),
      isPresent = _usePresence[0],
      safeToRemove = _usePresence[1];

  React.useEffect(function () {
    if (!isPresent && safeToRemove) {
      setTimeout(safeToRemove);
    }
  }, [isPresent, safeToRemove]);
  return /*#__PURE__*/React.createElement(FocusLock, {
    autoFocus: autoFocus,
    isDisabled: !trapFocus,
    initialFocusRef: initialFocusRef,
    finalFocusRef: finalFocusRef,
    restoreFocus: returnFocusOnClose,
    contentRef: dialogRef,
    lockFocusAcrossFrames: lockFocusAcrossFrames
  }, /*#__PURE__*/React.createElement(RemoveScroll, {
    removeScrollBar: !preserveScrollBarGap,
    allowPinchZoom: allowPinchZoom,
    enabled: blockScrollOnMount,
    forwardProps: true
  }, props.children));
}

/**
 * ModalOverlay renders a backdrop behind the modal. It is
 * also used as a wrapper for the modal content for better positioning.
 *
 * @see Docs https://chakra-ui.com/modal
 */
var ModalOverlay = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className;
      props.transition;
      var rest = _objectWithoutPropertiesLoose(props, _excluded2$1);

  var _className = cx("chakra-modal__overlay", className);

  var styles = useStyles();

  var overlayStyle = _extends({
    pos: "fixed",
    left: "0",
    top: "0",
    w: "100vw",
    h: "100vh"
  }, styles.overlay);

  var _useModalContext4 = useModalContext(),
      motionPreset = _useModalContext4.motionPreset;

  var motionProps = motionPreset === "none" ? {} : fadeConfig;
  return /*#__PURE__*/React.createElement(MotionDiv, _extends({}, motionProps, {
    __css: overlayStyle,
    ref: ref,
    className: _className
  }, rest));
});

if (__DEV__) {
  ModalOverlay.displayName = "ModalOverlay";
}

/**
 * ModalHeader
 *
 * React component that houses the title of the modal.
 *
 * @see Docs https://chakra-ui.com/modal
 */
var ModalHeader = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded3);

  var _useModalContext5 = useModalContext(),
      headerId = _useModalContext5.headerId,
      setHeaderMounted = _useModalContext5.setHeaderMounted;
  /**
   * Notify us if this component was rendered or used
   * so we can append `aria-labelledby` automatically
   */


  React.useEffect(function () {
    setHeaderMounted(true);
    return function () {
      return setHeaderMounted(false);
    };
  }, [setHeaderMounted]);

  var _className = cx("chakra-modal__header", className);

  var styles = useStyles();

  var headerStyles = _extends({
    flex: 0
  }, styles.header);

  return /*#__PURE__*/React.createElement(chakra.header, _extends({
    ref: ref,
    className: _className,
    id: headerId
  }, rest, {
    __css: headerStyles
  }));
});

if (__DEV__) {
  ModalHeader.displayName = "ModalHeader";
}

/**
 * ModalBody
 *
 * React component that houses the main content of the modal.
 *
 * @see Docs https://chakra-ui.com/modal
 */
var ModalBody = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded4);

  var _useModalContext6 = useModalContext(),
      bodyId = _useModalContext6.bodyId,
      setBodyMounted = _useModalContext6.setBodyMounted;
  /**
   * Notify us if this component was rendered or used
   * so we can append `aria-describedby` automatically
   */


  React.useEffect(function () {
    setBodyMounted(true);
    return function () {
      return setBodyMounted(false);
    };
  }, [setBodyMounted]);

  var _className = cx("chakra-modal__body", className);

  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    className: _className,
    id: bodyId
  }, rest, {
    __css: styles.body
  }));
});

if (__DEV__) {
  ModalBody.displayName = "ModalBody";
}

/**
 * ModalFooter houses the action buttons of the modal.
 * @see Docs https://chakra-ui.com/modal
 */
var ModalFooter = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded5);

  var _className = cx("chakra-modal__footer", className);

  var styles = useStyles();

  var footerStyles = _extends({
    display: "flex",
    alignItems: "center",
    justifyContent: "flex-end"
  }, styles.footer);

  return /*#__PURE__*/React.createElement(chakra.footer, _extends({
    ref: ref
  }, rest, {
    __css: footerStyles,
    className: _className
  }));
});

if (__DEV__) {
  ModalFooter.displayName = "ModalFooter";
}
/**
 * ModalCloseButton is used closes the modal.
 *
 * You don't need to pass the `onClick` to it, it reads the
 * `onClose` action from the modal context.
 */


var ModalCloseButton = /*#__PURE__*/forwardRef(function (props, ref) {
  var onClick = props.onClick,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded6);

  var _useModalContext7 = useModalContext(),
      onClose = _useModalContext7.onClose;

  var _className = cx("chakra-modal__close-btn", className);

  var styles = useStyles();
  return /*#__PURE__*/React.createElement(CloseButton, _extends({
    ref: ref,
    __css: styles.closeButton,
    className: _className,
    onClick: callAllHandlers(onClick, function (event) {
      event.stopPropagation();
      onClose();
    })
  }, rest));
});

if (__DEV__) {
  ModalCloseButton.displayName = "ModalCloseButton";
}

var _excluded$1 = ["leastDestructiveRef"];
function AlertDialog(props) {
  var leastDestructiveRef = props.leastDestructiveRef,
      rest = _objectWithoutPropertiesLoose(props, _excluded$1);

  return /*#__PURE__*/React.createElement(Modal, _extends({}, rest, {
    initialFocusRef: leastDestructiveRef
  }));
}
var AlertDialogContent = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(ModalContent, _extends({
    ref: ref,
    role: "alertdialog"
  }, props));
});

var _excluded = ["isOpen", "onClose", "placement", "children"],
    _excluded2 = ["className", "children"];

var _createContext = createContext(),
    DrawerContextProvider = _createContext[0],
    useDrawerContext = _createContext[1];

var placementMap = {
  start: {
    ltr: "left",
    rtl: "right"
  },
  end: {
    ltr: "right",
    rtl: "left"
  }
};

function getDrawerPlacement(placement, dir) {
  var _placementMap$placeme, _placementMap$placeme2;

  if (!placement) return;
  return (_placementMap$placeme = (_placementMap$placeme2 = placementMap[placement]) == null ? void 0 : _placementMap$placeme2[dir]) != null ? _placementMap$placeme : placement;
}

function Drawer(props) {
  var _theme$components;

  var isOpen = props.isOpen,
      onClose = props.onClose,
      _props$placement = props.placement,
      placementProp = _props$placement === void 0 ? "right" : _props$placement,
      children = props.children,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var theme = useTheme();
  var drawerStyleConfig = (_theme$components = theme.components) == null ? void 0 : _theme$components.Drawer;
  var placement = getDrawerPlacement(placementProp, theme.direction);
  return /*#__PURE__*/React.createElement(DrawerContextProvider, {
    value: {
      placement: placement
    }
  }, /*#__PURE__*/React.createElement(Modal, _extends({
    isOpen: isOpen,
    onClose: onClose,
    styleConfig: drawerStyleConfig
  }, rest), children));
}
var StyledSlide = chakra(Slide);

/**
 * ModalContent is used to group modal's content. It has all the
 * necessary `aria-*` properties to indicate that it is a modal
 */
var DrawerContent = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      children = props.children,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);

  var _useModalContext = useModalContext(),
      getDialogProps = _useModalContext.getDialogProps,
      getDialogContainerProps = _useModalContext.getDialogContainerProps,
      isOpen = _useModalContext.isOpen;

  var dialogProps = getDialogProps(rest, ref);
  var containerProps = getDialogContainerProps();

  var _className = cx("chakra-modal__content", className);

  var styles = useStyles();

  var dialogStyles = _extends({
    display: "flex",
    flexDirection: "column",
    position: "relative",
    width: "100%",
    outline: 0
  }, styles.dialog);

  var dialogContainerStyles = _extends({
    display: "flex",
    width: "100vw",
    height: "100vh",
    position: "fixed",
    left: 0,
    top: 0
  }, styles.dialogContainer);

  var _useDrawerContext = useDrawerContext(),
      placement = _useDrawerContext.placement;

  return /*#__PURE__*/React.createElement(chakra.div, _extends({}, containerProps, {
    className: "chakra-modal__content-container",
    __css: dialogContainerStyles
  }), /*#__PURE__*/React.createElement(ModalFocusScope, null, /*#__PURE__*/React.createElement(StyledSlide, _extends({
    direction: placement,
    "in": isOpen,
    className: _className
  }, dialogProps, {
    __css: dialogStyles
  }), children)));
});

if (__DEV__) {
  DrawerContent.displayName = "DrawerContent";
}

export { AlertDialog, ModalBody as AlertDialogBody, ModalCloseButton as AlertDialogCloseButton, AlertDialogContent, ModalFooter as AlertDialogFooter, ModalHeader as AlertDialogHeader, ModalOverlay as AlertDialogOverlay, Drawer, ModalBody as DrawerBody, ModalCloseButton as DrawerCloseButton, DrawerContent, ModalFooter as DrawerFooter, ModalHeader as DrawerHeader, ModalOverlay as DrawerOverlay, Modal, ModalBody, ModalCloseButton, ModalContent, ModalContextProvider, ModalFocusScope, ModalFooter, ModalHeader, ModalOverlay, useAriaHidden, useModal, useModalContext };
