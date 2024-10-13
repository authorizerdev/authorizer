'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var framerMotion = require('framer-motion');
var React = require('react');
var clickable = require('@chakra-ui/clickable');
var descendant = require('@chakra-ui/descendant');
var hooks = require('@chakra-ui/hooks');
var popper = require('@chakra-ui/popper');
var reactUtils = require('@chakra-ui/react-utils');

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

var _excluded$1 = ["id", "closeOnSelect", "closeOnBlur", "autoSelect", "isLazy", "isOpen", "defaultIsOpen", "onClose", "onOpen", "placement", "lazyBehavior", "direction", "computePositionOnMount"],
    _excluded2$1 = ["onMouseEnter", "onMouseMove", "onMouseLeave", "onClick", "isDisabled", "isFocusable", "closeOnSelect"],
    _excluded3$1 = ["type", "isChecked"],
    _excluded4$1 = ["children", "type", "value", "defaultValue", "onChange"];
/* -------------------------------------------------------------------------------------------------
 * Create context to track descendants and their indices
 * -----------------------------------------------------------------------------------------------*/

var _createDescendantCont = descendant.createDescendantContext(),
    MenuDescendantsProvider = _createDescendantCont[0],
    useMenuDescendantsContext = _createDescendantCont[1],
    useMenuDescendants = _createDescendantCont[2],
    useMenuDescendant = _createDescendantCont[3];

var _createContext = reactUtils.createContext({
  strict: false,
  name: "MenuContext"
}),
    MenuProvider = _createContext[0],
    useMenuContext = _createContext[1];

/**
 * React Hook to manage a menu
 *
 * It provides the logic and will be used with react context
 * to propagate its return value to all children
 */
function useMenu(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      id = _props.id,
      _props$closeOnSelect = _props.closeOnSelect,
      closeOnSelect = _props$closeOnSelect === void 0 ? true : _props$closeOnSelect,
      _props$closeOnBlur = _props.closeOnBlur,
      closeOnBlur = _props$closeOnBlur === void 0 ? true : _props$closeOnBlur,
      _props$autoSelect = _props.autoSelect,
      autoSelect = _props$autoSelect === void 0 ? true : _props$autoSelect,
      isLazy = _props.isLazy,
      isOpenProp = _props.isOpen,
      defaultIsOpen = _props.defaultIsOpen,
      onCloseProp = _props.onClose,
      onOpenProp = _props.onOpen,
      _props$placement = _props.placement,
      placement = _props$placement === void 0 ? "bottom-start" : _props$placement,
      _props$lazyBehavior = _props.lazyBehavior,
      lazyBehavior = _props$lazyBehavior === void 0 ? "unmount" : _props$lazyBehavior,
      direction = _props.direction,
      _props$computePositio = _props.computePositionOnMount,
      computePositionOnMount = _props$computePositio === void 0 ? false : _props$computePositio,
      popperProps = _objectWithoutPropertiesLoose(_props, _excluded$1);
  /**
   * Prepare the reference to the menu and disclosure
   */


  var menuRef = React__namespace.useRef(null);
  var buttonRef = React__namespace.useRef(null);
  /**
   * Context to register all menu item nodes
   */

  var descendants = useMenuDescendants();
  var focusMenu = React__namespace.useCallback(function () {
    utils.focus(menuRef.current, {
      nextTick: true,
      selectTextIfInput: false
    });
  }, []);
  var focusFirstItem = React__namespace.useCallback(function () {
    var id = setTimeout(function () {
      var first = descendants.firstEnabled();
      if (first) setFocusedIndex(first.index);
    });
    timeoutIds.current.add(id);
  }, [descendants]);
  var focusLastItem = React__namespace.useCallback(function () {
    var id = setTimeout(function () {
      var last = descendants.lastEnabled();
      if (last) setFocusedIndex(last.index);
    });
    timeoutIds.current.add(id);
  }, [descendants]);
  var onOpenInternal = React__namespace.useCallback(function () {
    onOpenProp == null ? void 0 : onOpenProp();

    if (autoSelect) {
      focusFirstItem();
    } else {
      focusMenu();
    }
  }, [autoSelect, focusFirstItem, focusMenu, onOpenProp]);

  var _useDisclosure = hooks.useDisclosure({
    isOpen: isOpenProp,
    defaultIsOpen: defaultIsOpen,
    onClose: onCloseProp,
    onOpen: onOpenInternal
  }),
      isOpen = _useDisclosure.isOpen,
      onOpen = _useDisclosure.onOpen,
      onClose = _useDisclosure.onClose,
      onToggle = _useDisclosure.onToggle;

  hooks.useOutsideClick({
    enabled: isOpen && closeOnBlur,
    ref: menuRef,
    handler: function handler(event) {
      var _buttonRef$current;

      if (!((_buttonRef$current = buttonRef.current) != null && _buttonRef$current.contains(event.target))) {
        onClose();
      }
    }
  });
  /**
   * Add some popper.js for dynamic positioning
   */

  var popper$1 = popper.usePopper(_extends({}, popperProps, {
    enabled: isOpen || computePositionOnMount,
    placement: placement,
    direction: direction
  }));

  var _React$useState = React__namespace.useState(-1),
      focusedIndex = _React$useState[0],
      setFocusedIndex = _React$useState[1];
  /**
   * Focus the button when we close the menu
   */


  hooks.useUpdateEffect(function () {
    if (!isOpen) {
      setFocusedIndex(-1);
    }
  }, [isOpen]);
  hooks.useFocusOnHide(menuRef, {
    focusRef: buttonRef,
    visible: isOpen,
    shouldFocus: true
  });
  /**
   * Generate unique ids for menu's list and button
   */

  var _useIds = hooks.useIds(id, "menu-button", "menu-list"),
      buttonId = _useIds[0],
      menuId = _useIds[1];

  var openAndFocusMenu = React__namespace.useCallback(function () {
    onOpen();
    focusMenu();
  }, [onOpen, focusMenu]);
  var timeoutIds = React__namespace.useRef(new Set([]));
  hooks.useUnmountEffect(function () {
    timeoutIds.current.forEach(function (id) {
      return clearTimeout(id);
    });
    timeoutIds.current.clear();
  });
  var openAndFocusFirstItem = React__namespace.useCallback(function () {
    onOpen();
    focusFirstItem();
  }, [focusFirstItem, onOpen]);
  var openAndFocusLastItem = React__namespace.useCallback(function () {
    onOpen();
    focusLastItem();
  }, [onOpen, focusLastItem]);
  var refocus = React__namespace.useCallback(function () {
    var _menuRef$current, _descendants$item;

    var doc = utils.getOwnerDocument(menuRef.current);
    var hasFocusWithin = (_menuRef$current = menuRef.current) == null ? void 0 : _menuRef$current.contains(doc.activeElement);
    var shouldRefocus = isOpen && !hasFocusWithin;
    if (!shouldRefocus) return;
    var node = (_descendants$item = descendants.item(focusedIndex)) == null ? void 0 : _descendants$item.node;

    if (node) {
      utils.focus(node, {
        selectTextIfInput: false,
        preventScroll: false
      });
    }
  }, [isOpen, focusedIndex, descendants]);
  return {
    openAndFocusMenu: openAndFocusMenu,
    openAndFocusFirstItem: openAndFocusFirstItem,
    openAndFocusLastItem: openAndFocusLastItem,
    onTransitionEnd: refocus,
    descendants: descendants,
    popper: popper$1,
    buttonId: buttonId,
    menuId: menuId,
    forceUpdate: popper$1.forceUpdate,
    orientation: "vertical",
    isOpen: isOpen,
    onToggle: onToggle,
    onOpen: onOpen,
    onClose: onClose,
    menuRef: menuRef,
    buttonRef: buttonRef,
    focusedIndex: focusedIndex,
    closeOnSelect: closeOnSelect,
    closeOnBlur: closeOnBlur,
    autoSelect: autoSelect,
    setFocusedIndex: setFocusedIndex,
    isLazy: isLazy,
    lazyBehavior: lazyBehavior
  };
}

/**
 * React Hook to manage a menu button.
 *
 * The assumption here is that the `useMenu` hook is used
 * in a component higher up the tree, and its return value
 * is passed as `context` to this hook.
 */
function useMenuButton(props, externalRef) {
  if (props === void 0) {
    props = {};
  }

  if (externalRef === void 0) {
    externalRef = null;
  }

  var menu = useMenuContext();
  var onToggle = menu.onToggle,
      popper = menu.popper,
      openAndFocusFirstItem = menu.openAndFocusFirstItem,
      openAndFocusLastItem = menu.openAndFocusLastItem;
  var onKeyDown = React__namespace.useCallback(function (event) {
    var eventKey = utils.normalizeEventKey(event);
    var keyMap = {
      Enter: openAndFocusFirstItem,
      ArrowDown: openAndFocusFirstItem,
      ArrowUp: openAndFocusLastItem
    };
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      event.stopPropagation();
      action(event);
    }
  }, [openAndFocusFirstItem, openAndFocusLastItem]);
  return _extends({}, props, {
    ref: reactUtils.mergeRefs(menu.buttonRef, externalRef, popper.referenceRef),
    id: menu.buttonId,
    "data-active": utils.dataAttr(menu.isOpen),
    "aria-expanded": menu.isOpen,
    "aria-haspopup": "menu",
    "aria-controls": menu.menuId,
    onClick: utils.callAllHandlers(props.onClick, onToggle),
    onKeyDown: utils.callAllHandlers(props.onKeyDown, onKeyDown)
  });
}

function isTargetMenuItem(target) {
  var _target$getAttribute;

  // this will catch `menuitem`, `menuitemradio`, `menuitemcheckbox`
  return utils.isHTMLElement(target) && !!((_target$getAttribute = target.getAttribute("role")) != null && _target$getAttribute.startsWith("menuitem"));
}
/* -------------------------------------------------------------------------------------------------
 * useMenuList
 * -----------------------------------------------------------------------------------------------*/


/**
 * React Hook to manage a menu list.
 *
 * The assumption here is that the `useMenu` hook is used
 * in a component higher up the tree, and its return value
 * is passed as `context` to this hook.
 */
function useMenuList(props, ref) {
  if (props === void 0) {
    props = {};
  }

  if (ref === void 0) {
    ref = null;
  }

  var menu = useMenuContext();

  if (!menu) {
    throw new Error("useMenuContext: context is undefined. Seems you forgot to wrap component within <Menu>");
  }

  var focusedIndex = menu.focusedIndex,
      setFocusedIndex = menu.setFocusedIndex,
      menuRef = menu.menuRef,
      isOpen = menu.isOpen,
      onClose = menu.onClose,
      menuId = menu.menuId,
      isLazy = menu.isLazy,
      lazyBehavior = menu.lazyBehavior;
  var descendants = useMenuDescendantsContext();
  /**
   * Hook that creates a keydown event handler that listens
   * to printable keyboard character press
   */

  var createTypeaheadHandler = hooks.useShortcut({
    preventDefault: function preventDefault(event) {
      return event.key !== " " && isTargetMenuItem(event.target);
    }
  });
  var onKeyDown = React__namespace.useCallback(function (event) {
    var eventKey = utils.normalizeEventKey(event);
    var keyMap = {
      Tab: function Tab(event) {
        return event.preventDefault();
      },
      Escape: onClose,
      ArrowDown: function ArrowDown() {
        var next = descendants.nextEnabled(focusedIndex);
        if (next) setFocusedIndex(next.index);
      },
      ArrowUp: function ArrowUp() {
        var prev = descendants.prevEnabled(focusedIndex);
        if (prev) setFocusedIndex(prev.index);
      }
    };
    var fn = keyMap[eventKey];

    if (fn) {
      event.preventDefault();
      fn(event);
      return;
    }
    /**
     * Typeahead: Based on current character pressed,
     * find the next item to be selected
     */


    var onTypeahead = createTypeaheadHandler(function (character) {
      var nextItem = utils.getNextItemFromSearch(descendants.values(), character, function (item) {
        var _item$node$textConten, _item$node;

        return (_item$node$textConten = item == null ? void 0 : (_item$node = item.node) == null ? void 0 : _item$node.textContent) != null ? _item$node$textConten : "";
      }, descendants.item(focusedIndex));

      if (nextItem) {
        var index = descendants.indexOf(nextItem.node);
        setFocusedIndex(index);
      }
    });

    if (isTargetMenuItem(event.target)) {
      onTypeahead(event);
    }
  }, [descendants, focusedIndex, createTypeaheadHandler, onClose, setFocusedIndex]);
  var hasBeenOpened = React__namespace.useRef(false);

  if (isOpen) {
    hasBeenOpened.current = true;
  }

  var shouldRenderChildren = utils.determineLazyBehavior({
    hasBeenSelected: hasBeenOpened.current,
    isLazy: isLazy,
    lazyBehavior: lazyBehavior,
    isSelected: isOpen
  });
  return _extends({}, props, {
    ref: reactUtils.mergeRefs(menuRef, ref),
    children: shouldRenderChildren ? props.children : null,
    tabIndex: -1,
    role: "menu",
    id: menuId,
    style: _extends({}, props.style, {
      transformOrigin: "var(--popper-transform-origin)"
    }),
    "aria-orientation": "vertical",
    onKeyDown: utils.callAllHandlers(props.onKeyDown, onKeyDown)
  });
}
/* -------------------------------------------------------------------------------------------------
 * useMenuPosition: Composes usePopper to position the menu
 * -----------------------------------------------------------------------------------------------*/

function useMenuPositioner(props) {
  if (props === void 0) {
    props = {};
  }

  var _useMenuContext = useMenuContext(),
      popper = _useMenuContext.popper,
      isOpen = _useMenuContext.isOpen;

  return popper.getPopperProps(_extends({}, props, {
    style: _extends({
      visibility: isOpen ? "visible" : "hidden"
    }, props.style)
  }));
}
/* -------------------------------------------------------------------------------------------------
 * useMenuItem: Hook for each menu item within the menu list.
   We also use it in `useMenuItemOption`
 * -----------------------------------------------------------------------------------------------*/

function useMenuItem(props, externalRef) {
  if (props === void 0) {
    props = {};
  }

  if (externalRef === void 0) {
    externalRef = null;
  }

  var _props2 = props,
      onMouseEnterProp = _props2.onMouseEnter,
      onMouseMoveProp = _props2.onMouseMove,
      onMouseLeaveProp = _props2.onMouseLeave,
      onClickProp = _props2.onClick,
      isDisabled = _props2.isDisabled,
      isFocusable = _props2.isFocusable,
      closeOnSelect = _props2.closeOnSelect,
      htmlProps = _objectWithoutPropertiesLoose(_props2, _excluded2$1);

  var menu = useMenuContext();
  var setFocusedIndex = menu.setFocusedIndex,
      focusedIndex = menu.focusedIndex,
      menuCloseOnSelect = menu.closeOnSelect,
      onClose = menu.onClose,
      menuRef = menu.menuRef,
      isOpen = menu.isOpen,
      menuId = menu.menuId;
  var ref = React__namespace.useRef(null);
  var id = menuId + "-menuitem-" + hooks.useId();
  /**
   * Register the menuitem's node into the domContext
   */

  var _useMenuDescendant = useMenuDescendant({
    disabled: isDisabled && !isFocusable
  }),
      index = _useMenuDescendant.index,
      register = _useMenuDescendant.register;

  var onMouseEnter = React__namespace.useCallback(function (event) {
    onMouseEnterProp == null ? void 0 : onMouseEnterProp(event);
    if (isDisabled) return;
    setFocusedIndex(index);
  }, [setFocusedIndex, index, isDisabled, onMouseEnterProp]);
  var onMouseMove = React__namespace.useCallback(function (event) {
    onMouseMoveProp == null ? void 0 : onMouseMoveProp(event);

    if (ref.current && !utils.isActiveElement(ref.current)) {
      onMouseEnter(event);
    }
  }, [onMouseEnter, onMouseMoveProp]);
  var onMouseLeave = React__namespace.useCallback(function (event) {
    onMouseLeaveProp == null ? void 0 : onMouseLeaveProp(event);
    if (isDisabled) return;
    setFocusedIndex(-1);
  }, [setFocusedIndex, isDisabled, onMouseLeaveProp]);
  var onClick = React__namespace.useCallback(function (event) {
    onClickProp == null ? void 0 : onClickProp(event);
    if (!isTargetMenuItem(event.currentTarget)) return;
    /**
     * Close menu and parent menus, allowing the MenuItem
     * to override its parent menu's `closeOnSelect` prop.
     */

    if (closeOnSelect != null ? closeOnSelect : menuCloseOnSelect) {
      onClose();
    }
  }, [onClose, onClickProp, menuCloseOnSelect, closeOnSelect]);
  var isFocused = index === focusedIndex;
  var trulyDisabled = isDisabled && !isFocusable;
  hooks.useUpdateEffect(function () {
    if (!isOpen) return;

    if (isFocused && !trulyDisabled && ref.current) {
      utils.focus(ref.current, {
        nextTick: true,
        selectTextIfInput: false,
        preventScroll: false
      });
    } else if (menuRef.current && !utils.isActiveElement(menuRef.current)) {
      utils.focus(menuRef.current, {
        preventScroll: false
      });
    }
  }, [isFocused, trulyDisabled, menuRef, isOpen]);
  var clickableProps = clickable.useClickable({
    onClick: onClick,
    onMouseEnter: onMouseEnter,
    onMouseMove: onMouseMove,
    onMouseLeave: onMouseLeave,
    ref: reactUtils.mergeRefs(register, ref, externalRef),
    isDisabled: isDisabled,
    isFocusable: isFocusable
  });
  return _extends({}, htmlProps, clickableProps, {
    id: id,
    role: "menuitem",
    tabIndex: isFocused ? 0 : -1
  });
}
/* -------------------------------------------------------------------------------------------------
 * useMenuOption: Composes useMenuItem to provide a selectable/checkable menu item
 * -----------------------------------------------------------------------------------------------*/

function useMenuOption(props, ref) {
  if (props === void 0) {
    props = {};
  }

  if (ref === void 0) {
    ref = null;
  }

  var _props3 = props,
      _props3$type = _props3.type,
      type = _props3$type === void 0 ? "radio" : _props3$type,
      isChecked = _props3.isChecked,
      rest = _objectWithoutPropertiesLoose(_props3, _excluded3$1);

  var ownProps = useMenuItem(rest, ref);
  return _extends({}, ownProps, {
    role: "menuitem" + type,
    "aria-checked": isChecked
  });
}
/* -------------------------------------------------------------------------------------------------
 * useMenuOptionGroup: Manages the state of multiple selectable menuitem or menu option
 * -----------------------------------------------------------------------------------------------*/

function useMenuOptionGroup(props) {
  if (props === void 0) {
    props = {};
  }

  var _props4 = props,
      children = _props4.children,
      _props4$type = _props4.type,
      type = _props4$type === void 0 ? "radio" : _props4$type,
      valueProp = _props4.value,
      defaultValue = _props4.defaultValue,
      onChangeProp = _props4.onChange,
      htmlProps = _objectWithoutPropertiesLoose(_props4, _excluded4$1);

  var isRadio = type === "radio";
  var fallback = isRadio ? "" : [];

  var _useControllableState = hooks.useControllableState({
    defaultValue: defaultValue != null ? defaultValue : fallback,
    value: valueProp,
    onChange: onChangeProp
  }),
      value = _useControllableState[0],
      setValue = _useControllableState[1];

  var onChange = React__namespace.useCallback(function (selectedValue) {
    if (type === "radio" && utils.isString(value)) {
      setValue(selectedValue);
    }

    if (type === "checkbox" && utils.isArray(value)) {
      var nextValue = value.includes(selectedValue) ? utils.removeItem(value, selectedValue) : utils.addItem(value, selectedValue);
      setValue(nextValue);
    }
  }, [value, setValue, type]);
  var validChildren = reactUtils.getValidChildren(children);
  var clones = validChildren.map(function (child) {
    /**
     * We've added an internal `id` to each `MenuItemOption`,
     * let's use that for type-checking.
     *
     * We can't rely on displayName or the element's type since
     * they can be changed by the user.
     */
    if (child.type.id !== "MenuItemOption") return child;

    var onClick = function onClick(event) {
      onChange(child.props.value);
      child.props.onClick == null ? void 0 : child.props.onClick(event);
    };

    var isChecked = type === "radio" ? child.props.value === value : value.includes(child.props.value);
    return /*#__PURE__*/React__namespace.cloneElement(child, {
      type: type,
      onClick: onClick,
      isChecked: isChecked
    });
  });
  return _extends({}, htmlProps, {
    children: clones
  });
}
function useMenuState() {
  var _useMenuContext2 = useMenuContext(),
      isOpen = _useMenuContext2.isOpen,
      onClose = _useMenuContext2.onClose;

  return {
    isOpen: isOpen,
    onClose: onClose
  };
}

var _excluded = ["descendants"],
    _excluded2 = ["children", "as"],
    _excluded3 = ["rootProps"],
    _excluded4 = ["type"],
    _excluded5 = ["icon", "iconSpacing", "command", "commandSpacing", "children"],
    _excluded6 = ["icon", "iconSpacing"],
    _excluded7 = ["className", "title"],
    _excluded8 = ["title", "children", "className"],
    _excluded9 = ["className", "children"],
    _excluded10 = ["className"];

/**
 * Menu provides context, state, and focus management
 * to its sub-components. It doesn't render any DOM node.
 */
var Menu = function Menu(props) {
  var children = props.children;
  var styles = system.useMultiStyleConfig("Menu", props);
  var ownProps = system.omitThemingProps(props);

  var _useTheme = system.useTheme(),
      direction = _useTheme.direction;

  var _useMenu = useMenu(_extends({}, ownProps, {
    direction: direction
  })),
      descendants = _useMenu.descendants,
      ctx = _objectWithoutPropertiesLoose(_useMenu, _excluded);

  var context = React__namespace.useMemo(function () {
    return ctx;
  }, [ctx]);
  var isOpen = context.isOpen,
      onClose = context.onClose,
      forceUpdate = context.forceUpdate;
  return /*#__PURE__*/React__namespace.createElement(MenuDescendantsProvider, {
    value: descendants
  }, /*#__PURE__*/React__namespace.createElement(MenuProvider, {
    value: context
  }, /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, utils.runIfFn(children, {
    isOpen: isOpen,
    onClose: onClose,
    forceUpdate: forceUpdate
  }))));
};

if (utils.__DEV__) {
  Menu.displayName = "Menu";
}

var StyledMenuButton = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.button, _extends({
    ref: ref
  }, props, {
    __css: _extends({
      display: "inline-flex",
      appearance: "none",
      alignItems: "center",
      outline: 0
    }, styles.button)
  }));
});
/**
 * The trigger for the menu list. Must be a direct child of `Menu`.
 */

var MenuButton = /*#__PURE__*/system.forwardRef(function (props, ref) {
  props.children;
      var As = props.as,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);

  var buttonProps = useMenuButton(rest, ref);
  var Element = As || StyledMenuButton;
  return /*#__PURE__*/React__namespace.createElement(Element, _extends({}, buttonProps, {
    className: utils.cx("chakra-menu__menu-button", props.className)
  }), /*#__PURE__*/React__namespace.createElement(system.chakra.span, {
    __css: {
      pointerEvents: "none",
      flex: "1 1 auto",
      minW: 0
    }
  }, props.children));
});

if (utils.__DEV__) {
  MenuButton.displayName = "MenuButton";
}

var motionVariants = {
  enter: {
    visibility: "visible",
    opacity: 1,
    scale: 1,
    transition: {
      duration: 0.2,
      ease: [0.4, 0, 0.2, 1]
    }
  },
  exit: {
    transitionEnd: {
      visibility: "hidden"
    },
    opacity: 0,
    scale: 0.8,
    transition: {
      duration: 0.1,
      easings: "easeOut"
    }
  }
}; // @future: only call `motion(chakra.div)` when we drop framer-motion v3 support

var MotionDiv = "custom" in framerMotion.motion ? framerMotion.motion.custom(system.chakra.div) : framerMotion.motion(system.chakra.div);
var MenuList = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var _props$zIndex, _styles$list;

  var rootProps = props.rootProps,
      rest = _objectWithoutPropertiesLoose(props, _excluded3);

  var _useMenuContext = useMenuContext(),
      isOpen = _useMenuContext.isOpen,
      onTransitionEnd = _useMenuContext.onTransitionEnd;

  var menulistProps = useMenuList(rest, ref);
  var positionerProps = useMenuPositioner(rootProps);
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, positionerProps, {
    __css: {
      zIndex: (_props$zIndex = props.zIndex) != null ? _props$zIndex : (_styles$list = styles.list) == null ? void 0 : _styles$list.zIndex
    }
  }), /*#__PURE__*/React__namespace.createElement(MotionDiv, _extends({}, menulistProps, {
    /**
     * We could call this on either `onAnimationComplete` or `onUpdate`.
     * It seems the re-focusing works better with the `onUpdate`
     */
    onUpdate: onTransitionEnd,
    className: utils.cx("chakra-menu__menu-list", menulistProps.className),
    variants: motionVariants,
    initial: false,
    animate: isOpen ? "enter" : "exit",
    __css: _extends({
      outline: 0
    }, styles.list)
  })));
});

if (utils.__DEV__) {
  MenuList.displayName = "MenuList";
}

var StyledMenuItem = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var type = props.type,
      rest = _objectWithoutPropertiesLoose(props, _excluded4);

  var styles = system.useStyles();
  /**
   * Given another component, use its type if present
   * Else, use no type to avoid invalid html, e.g. <a type="button" />
   * Else, fall back to "button"
   */

  var btnType = rest.as ? type != null ? type : undefined : "button";

  var buttonStyles = _extends({
    textDecoration: "none",
    color: "inherit",
    userSelect: "none",
    display: "flex",
    width: "100%",
    alignItems: "center",
    textAlign: "start",
    flex: "0 0 auto",
    outline: 0
  }, styles.item);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.button, _extends({
    ref: ref,
    type: btnType
  }, rest, {
    __css: buttonStyles
  }));
});
var MenuItem = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var icon = props.icon,
      _props$iconSpacing = props.iconSpacing,
      iconSpacing = _props$iconSpacing === void 0 ? "0.75rem" : _props$iconSpacing,
      command = props.command,
      _props$commandSpacing = props.commandSpacing,
      commandSpacing = _props$commandSpacing === void 0 ? "0.75rem" : _props$commandSpacing,
      children = props.children,
      rest = _objectWithoutPropertiesLoose(props, _excluded5);

  var menuitemProps = useMenuItem(rest, ref);
  var shouldWrap = icon || command;

  var _children = shouldWrap ? /*#__PURE__*/React__namespace.createElement("span", {
    style: {
      pointerEvents: "none",
      flex: 1
    }
  }, children) : children;

  return /*#__PURE__*/React__namespace.createElement(StyledMenuItem, _extends({}, menuitemProps, {
    className: utils.cx("chakra-menu__menuitem", menuitemProps.className)
  }), icon && /*#__PURE__*/React__namespace.createElement(MenuIcon, {
    fontSize: "0.8em",
    marginEnd: iconSpacing
  }, icon), _children, command && /*#__PURE__*/React__namespace.createElement(MenuCommand, {
    marginStart: commandSpacing
  }, command));
});

if (utils.__DEV__) {
  MenuItem.displayName = "MenuItem";
}

var CheckIcon = function CheckIcon(props) {
  return /*#__PURE__*/React__namespace.createElement("svg", _extends({
    viewBox: "0 0 14 14",
    width: "1em",
    height: "1em"
  }, props), /*#__PURE__*/React__namespace.createElement("polygon", {
    fill: "currentColor",
    points: "5.5 11.9993304 14 3.49933039 12.5 2 5.5 8.99933039 1.5 4.9968652 0 6.49933039"
  }));
};

var MenuItemOption = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var icon = props.icon,
      _props$iconSpacing2 = props.iconSpacing,
      iconSpacing = _props$iconSpacing2 === void 0 ? "0.75rem" : _props$iconSpacing2,
      rest = _objectWithoutPropertiesLoose(props, _excluded6);

  var optionProps = useMenuOption(rest, ref);
  return /*#__PURE__*/React__namespace.createElement(StyledMenuItem, _extends({}, optionProps, {
    className: utils.cx("chakra-menu__menuitem-option", rest.className)
  }), /*#__PURE__*/React__namespace.createElement(MenuIcon, {
    fontSize: "0.8em",
    marginEnd: iconSpacing,
    opacity: props.isChecked ? 1 : 0
  }, icon || /*#__PURE__*/React__namespace.createElement(CheckIcon, null)), /*#__PURE__*/React__namespace.createElement("span", {
    style: {
      flex: 1
    }
  }, optionProps.children));
});
MenuItemOption.id = "MenuItemOption";

if (utils.__DEV__) {
  MenuItemOption.displayName = "MenuItemOption";
}

var MenuOptionGroup = function MenuOptionGroup(props) {
  var className = props.className,
      title = props.title,
      rest = _objectWithoutPropertiesLoose(props, _excluded7);

  var ownProps = useMenuOptionGroup(rest);
  return /*#__PURE__*/React__namespace.createElement(MenuGroup, _extends({
    title: title,
    className: utils.cx("chakra-menu__option-group", className)
  }, ownProps));
};

if (utils.__DEV__) {
  MenuOptionGroup.displayName = "MenuOptionGroup";
}

var MenuGroup = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var title = props.title,
      children = props.children,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded8);

  var _className = utils.cx("chakra-menu__group__title", className);

  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement("div", {
    ref: ref,
    className: "chakra-menu__group",
    role: "group"
  }, title && /*#__PURE__*/React__namespace.createElement(system.chakra.p, _extends({
    className: _className
  }, rest, {
    __css: styles.groupTitle
  }), title), children);
});

if (utils.__DEV__) {
  MenuGroup.displayName = "MenuGroup";
}

var MenuCommand = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    ref: ref
  }, props, {
    __css: styles.command,
    className: "chakra-menu__command"
  }));
});

if (utils.__DEV__) {
  MenuCommand.displayName = "MenuCommand";
}

var MenuIcon = function MenuIcon(props) {
  var className = props.className,
      children = props.children,
      rest = _objectWithoutPropertiesLoose(props, _excluded9);

  var child = React__namespace.Children.only(children);
  var clone = /*#__PURE__*/React__namespace.isValidElement(child) ? /*#__PURE__*/React__namespace.cloneElement(child, {
    focusable: "false",
    "aria-hidden": true,
    className: utils.cx("chakra-menu__icon", child.props.className)
  }) : null;

  var _className = utils.cx("chakra-menu__icon-wrapper", className);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    className: _className
  }, rest, {
    __css: {
      flexShrink: 0
    }
  }), clone);
};

if (utils.__DEV__) {
  MenuIcon.displayName = "MenuIcon";
}

var MenuDivider = function MenuDivider(props) {
  var className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded10);

  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.hr, _extends({
    role: "separator",
    "aria-orientation": "horizontal",
    className: utils.cx("chakra-menu__divider", className)
  }, rest, {
    __css: styles.divider
  }));
};

if (utils.__DEV__) {
  MenuDivider.displayName = "MenuDivider";
}

exports.Menu = Menu;
exports.MenuButton = MenuButton;
exports.MenuCommand = MenuCommand;
exports.MenuDescendantsProvider = MenuDescendantsProvider;
exports.MenuDivider = MenuDivider;
exports.MenuGroup = MenuGroup;
exports.MenuIcon = MenuIcon;
exports.MenuItem = MenuItem;
exports.MenuItemOption = MenuItemOption;
exports.MenuList = MenuList;
exports.MenuOptionGroup = MenuOptionGroup;
exports.MenuProvider = MenuProvider;
exports.useMenu = useMenu;
exports.useMenuButton = useMenuButton;
exports.useMenuContext = useMenuContext;
exports.useMenuDescendant = useMenuDescendant;
exports.useMenuDescendants = useMenuDescendants;
exports.useMenuDescendantsContext = useMenuDescendantsContext;
exports.useMenuItem = useMenuItem;
exports.useMenuList = useMenuList;
exports.useMenuOption = useMenuOption;
exports.useMenuOptionGroup = useMenuOptionGroup;
exports.useMenuPositioner = useMenuPositioner;
exports.useMenuState = useMenuState;
