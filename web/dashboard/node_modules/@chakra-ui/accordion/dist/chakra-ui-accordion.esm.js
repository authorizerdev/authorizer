import { Icon } from '@chakra-ui/icon';
import { forwardRef, useMultiStyleConfig, omitThemingProps, StylesProvider, chakra, useStyles } from '@chakra-ui/system';
import { Collapse } from '@chakra-ui/transition';
import { normalizeEventKey, callAllHandlers, isUndefined, isArray, warn, addItem, removeItem, focus, cx, __DEV__, runIfFn } from '@chakra-ui/utils';
import { createContext, mergeRefs } from '@chakra-ui/react-utils';
import * as React from 'react';
import { useState, useRef, useCallback } from 'react';
import { createDescendantContext } from '@chakra-ui/descendant';
import { useUnmountEffect, useControllableState, useIds } from '@chakra-ui/hooks';

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

var _excluded$1 = ["onChange", "defaultIndex", "index", "allowMultiple", "allowToggle"],
    _excluded2$1 = ["isDisabled", "isFocusable", "id"];
/* -------------------------------------------------------------------------------------------------
 * Create context to track descendants and their indices
 * -----------------------------------------------------------------------------------------------*/

var _createDescendantCont = createDescendantContext(),
    AccordionDescendantsProvider = _createDescendantCont[0],
    useAccordionDescendantsContext = _createDescendantCont[1],
    useAccordionDescendants = _createDescendantCont[2],
    useAccordionDescendant = _createDescendantCont[3];

/**
 * useAccordion hook provides all the state and focus management logic
 * for accordion items.
 */
function useAccordion(props) {
  var onChange = props.onChange,
      defaultIndex = props.defaultIndex,
      indexProp = props.index,
      allowMultiple = props.allowMultiple,
      allowToggle = props.allowToggle,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded$1); // validate the props and `warn` if used incorrectly


  allowMultipleWarning(props);
  allowMultipleAndAllowToggleWarning(props);
  /**
   * Think of this as the register to each accordion item.
   * We used to manage focus between accordion item buttons.
   *
   * Every accordion item, registers their button refs in this context
   */

  var descendants = useAccordionDescendants();
  /**
   * This state is used to track the index focused accordion
   * button when click on the button, tab on the button, or
   * use the down/up arrow to navigate.
   */

  var _useState = useState(-1),
      focusedIndex = _useState[0],
      setFocusedIndex = _useState[1];
  /**
   * Reset focused index when accordion unmounts
   * or descendants change
   */


  useUnmountEffect(function () {
    setFocusedIndex(-1);
  });
  /**
   * Hook that manages the controlled and un-controlled state
   * for the accordion.
   */

  var _useControllableState = useControllableState({
    value: indexProp,
    defaultValue: function defaultValue() {
      if (allowMultiple) return defaultIndex != null ? defaultIndex : [];
      return defaultIndex != null ? defaultIndex : -1;
    },
    onChange: onChange
  }),
      index = _useControllableState[0],
      setIndex = _useControllableState[1];
  /**
   * Gets the `isOpen` and `onChange` props for a child accordion item based on
   * the child's index.
   *
   * @param idx {number} The index of the child accordion item
   */


  var getAccordionItemProps = function getAccordionItemProps(idx) {
    var isOpen = false;

    if (idx !== null) {
      isOpen = isArray(index) ? index.includes(idx) : index === idx;
    }

    var onChange = function onChange(isOpen) {
      if (idx === null) return;

      if (allowMultiple && isArray(index)) {
        var nextState = isOpen ? addItem(index, idx) : removeItem(index, idx);
        setIndex(nextState);
      } else if (isOpen) {
        setIndex(idx);
      } else if (allowToggle) {
        setIndex(-1);
      }
    };

    return {
      isOpen: isOpen,
      onChange: onChange
    };
  };

  return {
    index: index,
    setIndex: setIndex,
    htmlProps: htmlProps,
    getAccordionItemProps: getAccordionItemProps,
    focusedIndex: focusedIndex,
    setFocusedIndex: setFocusedIndex,
    descendants: descendants
  };
}

var _createContext$1 = createContext({
  name: "AccordionContext",
  errorMessage: "useAccordionContext: `context` is undefined. Seems you forgot to wrap the accordion components in `<Accordion />`"
}),
    AccordionProvider = _createContext$1[0],
    useAccordionContext = _createContext$1[1];

/**
 * useAccordionItem
 *
 * React hook that provides the open/close functionality
 * for an accordion item and its children
 */
function useAccordionItem(props) {
  var isDisabled = props.isDisabled,
      isFocusable = props.isFocusable,
      id = props.id,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded2$1);

  var _useAccordionContext = useAccordionContext(),
      getAccordionItemProps = _useAccordionContext.getAccordionItemProps,
      setFocusedIndex = _useAccordionContext.setFocusedIndex;

  var buttonRef = useRef(null);
  /**
   * Generate unique ids for all accordion item components (button and panel)
   */

  var _useIds = useIds(id, "accordion-button", "accordion-panel"),
      buttonId = _useIds[0],
      panelId = _useIds[1];

  focusableNotDisabledWarning(props);
  /**
   * Think of this as a way to register this accordion item
   * with its parent `useAccordion`
   */

  var _useAccordionDescenda = useAccordionDescendant({
    disabled: isDisabled && !isFocusable
  }),
      register = _useAccordionDescenda.register,
      index = _useAccordionDescenda.index,
      descendants = _useAccordionDescenda.descendants;

  var _getAccordionItemProp = getAccordionItemProps(index === -1 ? null : index),
      isOpen = _getAccordionItemProp.isOpen,
      onChange = _getAccordionItemProp.onChange;

  warnIfOpenAndDisabled({
    isOpen: isOpen,
    isDisabled: isDisabled
  });

  var onOpen = function onOpen() {
    onChange == null ? void 0 : onChange(true);
  };

  var onClose = function onClose() {
    onChange == null ? void 0 : onChange(false);
  };
  /**
   * Toggle the visibility of the accordion item
   */


  var onClick = useCallback(function () {
    onChange == null ? void 0 : onChange(!isOpen);
    setFocusedIndex(index);
  }, [index, setFocusedIndex, isOpen, onChange]);
  /**
   * Manage keyboard navigation between accordion items.
   */

  var onKeyDown = useCallback(function (event) {
    var eventKey = normalizeEventKey(event);
    var keyMap = {
      ArrowDown: function ArrowDown() {
        var next = descendants.nextEnabled(index);
        if (next) focus(next.node);
      },
      ArrowUp: function ArrowUp() {
        var prev = descendants.prevEnabled(index);
        if (prev) focus(prev.node);
      },
      Home: function Home() {
        var first = descendants.firstEnabled();
        if (first) focus(first.node);
      },
      End: function End() {
        var last = descendants.lastEnabled();
        if (last) focus(last.node);
      }
    };
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      action(event);
    }
  }, [descendants, index]);
  /**
   * Since each accordion item's button still remains tabbable, let's
   * update the focusedIndex when it receives focus
   */

  var onFocus = useCallback(function () {
    setFocusedIndex(index);
  }, [setFocusedIndex, index]);
  var getButtonProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      type: "button",
      ref: mergeRefs(register, buttonRef, ref),
      id: buttonId,
      disabled: !!isDisabled,
      "aria-expanded": !!isOpen,
      "aria-controls": panelId,
      onClick: callAllHandlers(props.onClick, onClick),
      onFocus: callAllHandlers(props.onFocus, onFocus),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown)
    });
  }, [buttonId, isDisabled, isOpen, onClick, onFocus, onKeyDown, panelId, register]);
  var getPanelProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      ref: ref,
      role: "region",
      id: panelId,
      "aria-labelledby": buttonId,
      hidden: !isOpen
    });
  }, [buttonId, isOpen, panelId]);
  return {
    isOpen: isOpen,
    isDisabled: isDisabled,
    isFocusable: isFocusable,
    onOpen: onOpen,
    onClose: onClose,
    getButtonProps: getButtonProps,
    getPanelProps: getPanelProps,
    htmlProps: htmlProps
  };
}

/* -------------------------------------------------------------------------------------------------
 * Validate accordion and accordion item props, and emit warnings.
 * -----------------------------------------------------------------------------------------------*/
function allowMultipleWarning(props) {
  var index = props.index || props.defaultIndex;
  var condition = !isUndefined(index) && !isArray(index) && props.allowMultiple;
  warn({
    condition: !!condition,
    message: "If 'allowMultiple' is passed, then 'index' or 'defaultIndex' must be an array. You passed: " + typeof index + ","
  });
}

function allowMultipleAndAllowToggleWarning(props) {
  warn({
    condition: !!(props.allowMultiple && props.allowToggle),
    message: "If 'allowMultiple' is passed, 'allowToggle' will be ignored. Either remove 'allowToggle' or 'allowMultiple' depending on whether you want multiple accordions visible or not"
  });
}

function focusableNotDisabledWarning(props) {
  warn({
    condition: !!(props.isFocusable && !props.isDisabled),
    message: "Using only 'isFocusable', this prop is reserved for situations where you pass 'isDisabled' but you still want the element to receive focus (A11y). Either remove it or pass 'isDisabled' as well.\n    "
  });
}

function warnIfOpenAndDisabled(props) {
  warn({
    condition: props.isOpen && !!props.isDisabled,
    message: "Cannot open a disabled accordion item"
  });
}

var _excluded = ["children", "reduceMotion"],
    _excluded2 = ["htmlProps", "descendants"],
    _excluded3 = ["htmlProps"];
/* -------------------------------------------------------------------------------------------------
 * Accordion - The wrapper that provides context for all accordion items
 * -----------------------------------------------------------------------------------------------*/

/**
 * The wrapper that provides context and focus management
 * for all accordion items.
 *
 * It wraps all accordion items in a `div` for better grouping.
 * @see Docs https://chakra-ui.com/accordion
 */
var Accordion = /*#__PURE__*/forwardRef(function (_ref, ref) {
  var children = _ref.children,
      reduceMotion = _ref.reduceMotion,
      props = _objectWithoutPropertiesLoose(_ref, _excluded);

  var styles = useMultiStyleConfig("Accordion", props);
  var ownProps = omitThemingProps(props);

  var _useAccordion = useAccordion(ownProps),
      htmlProps = _useAccordion.htmlProps,
      descendants = _useAccordion.descendants,
      context = _objectWithoutPropertiesLoose(_useAccordion, _excluded2);

  var ctx = React.useMemo(function () {
    return _extends({}, context, {
      reduceMotion: !!reduceMotion
    });
  }, [context, reduceMotion]);
  return /*#__PURE__*/React.createElement(AccordionDescendantsProvider, {
    value: descendants
  }, /*#__PURE__*/React.createElement(AccordionProvider, {
    value: ctx
  }, /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref
  }, htmlProps, {
    className: cx("chakra-accordion", props.className)
  }), children))));
});

if (__DEV__) {
  Accordion.displayName = "Accordion";
}
/* -------------------------------------------------------------------------------------------------
 * Accordion Item
 * -----------------------------------------------------------------------------------------------*/


var _createContext = createContext({
  name: "AccordionItemContext",
  errorMessage: "useAccordionItemContext: `context` is undefined. Seems you forgot to wrap the accordion item parts in `<AccordionItem />` "
}),
    AccordionItemProvider = _createContext[0],
    useAccordionItemContext = _createContext[1];

/**
 * AccordionItem is a single accordion that provides the open-close
 * behavior when the accordion button is clicked.
 *
 * It also provides context for the accordion button and panel.
 */
var AccordionItem = /*#__PURE__*/forwardRef(function (props, ref) {
  var children = props.children,
      className = props.className;

  var _useAccordionItem = useAccordionItem(props),
      htmlProps = _useAccordionItem.htmlProps,
      context = _objectWithoutPropertiesLoose(_useAccordionItem, _excluded3);

  var styles = useStyles();

  var containerStyles = _extends({}, styles.container, {
    overflowAnchor: "none"
  });

  var ctx = React.useMemo(function () {
    return context;
  }, [context]);
  return /*#__PURE__*/React.createElement(AccordionItemProvider, {
    value: ctx
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref
  }, htmlProps, {
    className: cx("chakra-accordion__item", className),
    __css: containerStyles
  }), runIfFn(children, {
    isExpanded: !!context.isOpen,
    isDisabled: !!context.isDisabled
  })));
});

if (__DEV__) {
  AccordionItem.displayName = "AccordionItem";
}
/**
 * React hook to get the state and actions of an accordion item
 */


function useAccordionItemState() {
  var _useAccordionItemCont = useAccordionItemContext(),
      isOpen = _useAccordionItemCont.isOpen,
      isDisabled = _useAccordionItemCont.isDisabled,
      onClose = _useAccordionItemCont.onClose,
      onOpen = _useAccordionItemCont.onOpen;

  return {
    isOpen: isOpen,
    onClose: onClose,
    isDisabled: isDisabled,
    onOpen: onOpen
  };
}
/* -------------------------------------------------------------------------------------------------
 * Accordion Item => Button
 * -----------------------------------------------------------------------------------------------*/

/**
 * AccordionButton is used expands and collapses an accordion item.
 * It must be a child of `AccordionItem`.
 *
 * Note 🚨: Each accordion button must be wrapped in an heading tag,
 * that is appropriate for the information architecture of the page.
 */
var AccordionButton = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useAccordionItemCont2 = useAccordionItemContext(),
      getButtonProps = _useAccordionItemCont2.getButtonProps;

  var buttonProps = getButtonProps(props, ref);
  var styles = useStyles();

  var buttonStyles = _extends({
    display: "flex",
    alignItems: "center",
    width: "100%",
    outline: 0
  }, styles.button);

  return /*#__PURE__*/React.createElement(chakra.button, _extends({}, buttonProps, {
    className: cx("chakra-accordion__button", props.className),
    __css: buttonStyles
  }));
});

if (__DEV__) {
  AccordionButton.displayName = "AccordionButton";
}
/* -------------------------------------------------------------------------------------------------
 * Accordion Item => Panel
 * -----------------------------------------------------------------------------------------------*/


/**
 * Accordion panel that holds the content for each accordion.
 * It shows and hides based on the state login from the `AccordionItem`.
 *
 * It uses the `Collapse` component to animate its height.
 */
var AccordionPanel = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useAccordionContext = useAccordionContext(),
      reduceMotion = _useAccordionContext.reduceMotion;

  var _useAccordionItemCont3 = useAccordionItemContext(),
      getPanelProps = _useAccordionItemCont3.getPanelProps,
      isOpen = _useAccordionItemCont3.isOpen; // remove `hidden` prop, 'coz we're using height animation


  var panelProps = getPanelProps(props, ref);

  var _className = cx("chakra-accordion__panel", props.className);

  var styles = useStyles();

  if (!reduceMotion) {
    delete panelProps.hidden;
  }

  var child = /*#__PURE__*/React.createElement(chakra.div, _extends({}, panelProps, {
    __css: styles.panel,
    className: _className
  }));

  if (!reduceMotion) {
    return /*#__PURE__*/React.createElement(Collapse, {
      "in": isOpen
    }, child);
  }

  return child;
});

if (__DEV__) {
  AccordionPanel.displayName = "AccordionPanel";
}
/* -------------------------------------------------------------------------------------------------
 * Accordion Item => Icon
 * -----------------------------------------------------------------------------------------------*/

/**
 * AccordionIcon that gives a visual cue of the open/close state of the accordion item.
 * It rotates `180deg` based on the open/close state.
 */


var AccordionIcon = function AccordionIcon(props) {
  var _useAccordionItemCont4 = useAccordionItemContext(),
      isOpen = _useAccordionItemCont4.isOpen,
      isDisabled = _useAccordionItemCont4.isDisabled;

  var _useAccordionContext2 = useAccordionContext(),
      reduceMotion = _useAccordionContext2.reduceMotion;

  var _className = cx("chakra-accordion__icon", props.className);

  var styles = useStyles();

  var iconStyles = _extends({
    opacity: isDisabled ? 0.4 : 1,
    transform: isOpen ? "rotate(-180deg)" : undefined,
    transition: reduceMotion ? undefined : "transform 0.2s",
    transformOrigin: "center"
  }, styles.icon);

  return /*#__PURE__*/React.createElement(Icon, _extends({
    viewBox: "0 0 24 24",
    "aria-hidden": true,
    className: _className,
    __css: iconStyles
  }, props), /*#__PURE__*/React.createElement("path", {
    fill: "currentColor",
    d: "M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z"
  }));
};

if (__DEV__) {
  AccordionIcon.displayName = "AccordionIcon";
}

export { Accordion, AccordionButton, AccordionDescendantsProvider, AccordionIcon, AccordionItem, AccordionPanel, AccordionProvider, useAccordion, useAccordionContext, useAccordionDescendant, useAccordionDescendants, useAccordionDescendantsContext, useAccordionItem, useAccordionItemState };
