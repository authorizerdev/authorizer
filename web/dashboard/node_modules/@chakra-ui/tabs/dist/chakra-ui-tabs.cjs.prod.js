'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var React = require('react');
var clickable = require('@chakra-ui/clickable');
var descendant = require('@chakra-ui/descendant');
var hooks = require('@chakra-ui/hooks');
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

var _excluded$1 = ["defaultIndex", "onChange", "index", "isManual", "isLazy", "lazyBehavior", "orientation", "direction"],
    _excluded2$1 = ["isDisabled", "isFocusable"],
    _excluded3 = ["isSelected", "id", "children"];
/* -------------------------------------------------------------------------------------------------
 * Create context to track descendants and their indices
 * -----------------------------------------------------------------------------------------------*/

var _createDescendantCont = descendant.createDescendantContext(),
    TabsDescendantsProvider = _createDescendantCont[0],
    useTabsDescendantsContext = _createDescendantCont[1],
    useTabsDescendants = _createDescendantCont[2],
    useTabsDescendant = _createDescendantCont[3];

/**
 * Tabs hooks that provides all the states, and accessibility
 * helpers to keep all things working properly.
 *
 * Its returned object will be passed unto a Context Provider
 * so all child components can read from it.
 * There is no document link yet
 * @see Docs https://chakra-ui.com/docs/components/useTabs
 */
function useTabs(props) {
  var defaultIndex = props.defaultIndex,
      onChange = props.onChange,
      index = props.index,
      isManual = props.isManual,
      isLazy = props.isLazy,
      _props$lazyBehavior = props.lazyBehavior,
      lazyBehavior = _props$lazyBehavior === void 0 ? "unmount" : _props$lazyBehavior,
      _props$orientation = props.orientation,
      orientation = _props$orientation === void 0 ? "horizontal" : _props$orientation,
      _props$direction = props.direction,
      direction = _props$direction === void 0 ? "ltr" : _props$direction,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded$1);
  /**
   * We use this to keep track of the index of the focused tab.
   *
   * Tabs can be automatically activated, this means selection follows focus.
   * When we navigate with the arrow keys, we move focus and selection to next/prev tab
   *
   * Tabs can also be manually activated, this means selection does not follow focus.
   * When we navigate with the arrow keys, we only move focus NOT selection. The user
   * will need not manually activate the tab using `Enter` or `Space`.
   *
   * This is why we need to keep track of the `focusedIndex` and `selectedIndex`
   */


  var _React$useState = React__namespace.useState(defaultIndex != null ? defaultIndex : 0),
      focusedIndex = _React$useState[0],
      setFocusedIndex = _React$useState[1];

  var _useControllableState = hooks.useControllableState({
    defaultValue: defaultIndex != null ? defaultIndex : 0,
    value: index,
    onChange: onChange
  }),
      selectedIndex = _useControllableState[0],
      setSelectedIndex = _useControllableState[1];
  /**
   * Sync focused `index` with controlled `selectedIndex` (which is the `props.index`)
   */


  React__namespace.useEffect(function () {
    if (index != null) {
      setFocusedIndex(index);
    }
  }, [index]);
  /**
   * Think of `useDescendants` as a register for the tab nodes.
   */

  var descendants = useTabsDescendants();
  /**
   * Generate a unique id or use user-provided id for the tabs widget
   */

  var id = hooks.useId(props.id, "tabs");
  return {
    id: id,
    selectedIndex: selectedIndex,
    focusedIndex: focusedIndex,
    setSelectedIndex: setSelectedIndex,
    setFocusedIndex: setFocusedIndex,
    isManual: isManual,
    isLazy: isLazy,
    lazyBehavior: lazyBehavior,
    orientation: orientation,
    descendants: descendants,
    direction: direction,
    htmlProps: htmlProps
  };
}

var _createContext = reactUtils.createContext({
  name: "TabsContext",
  errorMessage: "useTabsContext: `context` is undefined. Seems you forgot to wrap all tabs components within <Tabs />"
}),
    TabsProvider = _createContext[0],
    useTabsContext = _createContext[1];

/**
 * Tabs hook to manage multiple tab buttons,
 * and ensures only one tab is selected per time.
 *
 * @param props props object for the tablist
 */
function useTabList(props) {
  var _useTabsContext = useTabsContext(),
      focusedIndex = _useTabsContext.focusedIndex,
      orientation = _useTabsContext.orientation,
      direction = _useTabsContext.direction;

  var descendants = useTabsDescendantsContext();
  var onKeyDown = React__namespace.useCallback(function (event) {
    var _keyMap;

    var nextTab = function nextTab() {
      var next = descendants.nextEnabled(focusedIndex);
      if (next) utils.focus(next.node);
    };

    var prevTab = function prevTab() {
      var prev = descendants.prevEnabled(focusedIndex);
      if (prev) utils.focus(prev.node);
    };

    var firstTab = function firstTab() {
      var first = descendants.firstEnabled();
      if (first) utils.focus(first.node);
    };

    var lastTab = function lastTab() {
      var last = descendants.lastEnabled();
      if (last) utils.focus(last.node);
    };

    var isHorizontal = orientation === "horizontal";
    var isVertical = orientation === "vertical";
    var eventKey = utils.normalizeEventKey(event);
    var ArrowStart = direction === "ltr" ? "ArrowLeft" : "ArrowRight";
    var ArrowEnd = direction === "ltr" ? "ArrowRight" : "ArrowLeft";
    var keyMap = (_keyMap = {}, _keyMap[ArrowStart] = function () {
      return isHorizontal && prevTab();
    }, _keyMap[ArrowEnd] = function () {
      return isHorizontal && nextTab();
    }, _keyMap.ArrowDown = function ArrowDown() {
      return isVertical && nextTab();
    }, _keyMap.ArrowUp = function ArrowUp() {
      return isVertical && prevTab();
    }, _keyMap.Home = firstTab, _keyMap.End = lastTab, _keyMap);
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      action(event);
    }
  }, [descendants, focusedIndex, orientation, direction]);
  return _extends({}, props, {
    role: "tablist",
    "aria-orientation": orientation,
    onKeyDown: utils.callAllHandlers(props.onKeyDown, onKeyDown)
  });
}

/**
 * Tabs hook to manage each tab button.
 *
 * A tab can be disabled and focusable, or both,
 * hence the use of `useClickable` to handle this scenario
 */
function useTab(props) {
  var isDisabled = props.isDisabled,
      isFocusable = props.isFocusable,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded2$1);

  var _useTabsContext2 = useTabsContext(),
      setSelectedIndex = _useTabsContext2.setSelectedIndex,
      isManual = _useTabsContext2.isManual,
      id = _useTabsContext2.id,
      setFocusedIndex = _useTabsContext2.setFocusedIndex,
      selectedIndex = _useTabsContext2.selectedIndex;

  var _useTabsDescendant = useTabsDescendant({
    disabled: isDisabled && !isFocusable
  }),
      index = _useTabsDescendant.index,
      register = _useTabsDescendant.register;

  var isSelected = index === selectedIndex;

  var onClick = function onClick() {
    setSelectedIndex(index);
  };

  var onFocus = function onFocus() {
    setFocusedIndex(index);
    var isDisabledButFocusable = isDisabled && isFocusable;
    var shouldSelect = !isManual && !isDisabledButFocusable;

    if (shouldSelect) {
      setSelectedIndex(index);
    }
  };

  var clickableProps = clickable.useClickable(_extends({}, htmlProps, {
    ref: reactUtils.mergeRefs(register, props.ref),
    isDisabled: isDisabled,
    isFocusable: isFocusable,
    onClick: utils.callAllHandlers(props.onClick, onClick)
  }));
  var type = "button";
  return _extends({}, clickableProps, {
    id: makeTabId(id, index),
    role: "tab",
    tabIndex: isSelected ? 0 : -1,
    type: type,
    "aria-selected": isSelected,
    "aria-controls": makeTabPanelId(id, index),
    onFocus: isDisabled ? undefined : utils.callAllHandlers(props.onFocus, onFocus)
  });
}

/**
 * Tabs hook for managing the visibility of multiple tab panels.
 *
 * Since only one panel can be show at a time, we use `cloneElement`
 * to inject `selected` panel to each TabPanel.
 *
 * It returns a cloned version of its children with
 * all functionality included.
 */
function useTabPanels(props) {
  var context = useTabsContext();
  var id = context.id,
      selectedIndex = context.selectedIndex;
  var validChildren = reactUtils.getValidChildren(props.children);
  var children = validChildren.map(function (child, index) {
    return /*#__PURE__*/React__namespace.cloneElement(child, {
      isSelected: index === selectedIndex,
      id: makeTabPanelId(id, index),
      // Refers to the associated tab element, and also provides an accessible name to the tab panel.
      "aria-labelledby": makeTabId(id, index)
    });
  });
  return _extends({}, props, {
    children: children
  });
}
/**
 * Tabs hook for managing the visible/hidden states
 * of the tab panel.
 *
 * @param props props object for the tab panel
 */

function useTabPanel(props) {
  var isSelected = props.isSelected,
      id = props.id,
      children = props.children,
      htmlProps = _objectWithoutPropertiesLoose(props, _excluded3);

  var _useTabsContext3 = useTabsContext(),
      isLazy = _useTabsContext3.isLazy,
      lazyBehavior = _useTabsContext3.lazyBehavior;

  var hasBeenSelected = React__namespace.useRef(false);

  if (isSelected) {
    hasBeenSelected.current = true;
  }

  var shouldRenderChildren = utils.determineLazyBehavior({
    hasBeenSelected: hasBeenSelected.current,
    isSelected: isSelected,
    isLazy: isLazy,
    lazyBehavior: lazyBehavior
  });
  return _extends({
    // Puts the tabpanel in the page `Tab` sequence.
    tabIndex: 0
  }, htmlProps, {
    children: shouldRenderChildren ? children : null,
    role: "tabpanel",
    hidden: !isSelected,
    id: id
  });
}
/**
 * Tabs hook to show an animated indicators that
 * follows the active tab.
 *
 * The way we do it is by measuring the DOM Rect (or dimensions)
 * of the active tab, and return that as CSS style for
 * the indicator.
 */

function useTabIndicator() {
  var context = useTabsContext();
  var descendants = useTabsDescendantsContext();
  var selectedIndex = context.selectedIndex,
      orientation = context.orientation;
  var isHorizontal = orientation === "horizontal";
  var isVertical = orientation === "vertical"; // Get the clientRect of the selected tab

  var _React$useState2 = React__namespace.useState(function () {
    if (isHorizontal) return {
      left: 0,
      width: 0
    };
    if (isVertical) return {
      top: 0,
      height: 0
    };
    return undefined;
  }),
      rect = _React$useState2[0],
      setRect = _React$useState2[1];

  var _React$useState3 = React__namespace.useState(false),
      hasMeasured = _React$useState3[0],
      setHasMeasured = _React$useState3[1]; // Update the selected tab rect when the selectedIndex changes


  hooks.useSafeLayoutEffect(function () {
    if (utils.isUndefined(selectedIndex)) return undefined;
    var tab = descendants.item(selectedIndex);
    if (utils.isUndefined(tab)) return undefined; // Horizontal Tab: Calculate width and left distance

    if (isHorizontal) {
      setRect({
        left: tab.node.offsetLeft,
        width: tab.node.offsetWidth
      });
    } // Vertical Tab: Calculate height and top distance


    if (isVertical) {
      setRect({
        top: tab.node.offsetTop,
        height: tab.node.offsetHeight
      });
    } // Prevent unwanted transition from 0 to measured rect
    // by setting the measured state in the next tick


    var id = requestAnimationFrame(function () {
      setHasMeasured(true);
    });
    return function () {
      if (id) {
        cancelAnimationFrame(id);
      }
    };
  }, [selectedIndex, isHorizontal, isVertical, descendants]);
  return _extends({
    position: "absolute",
    transitionProperty: "left, right, top, bottom",
    transitionDuration: hasMeasured ? "200ms" : "0ms",
    transitionTimingFunction: "cubic-bezier(0, 0, 0.2, 1)"
  }, rect);
}

function makeTabId(id, index) {
  return id + "--tab-" + index;
}

function makeTabPanelId(id, index) {
  return id + "--tabpanel-" + index;
}

var _excluded = ["children", "className"],
    _excluded2 = ["htmlProps", "descendants"];

/**
 * Tabs
 *
 * Provides context and logic for all tabs components.
 */
var Tabs = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("Tabs", props);

  var _omitThemingProps = system.omitThemingProps(props),
      children = _omitThemingProps.children,
      className = _omitThemingProps.className,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded);

  var _useTabs = useTabs(rest),
      htmlProps = _useTabs.htmlProps,
      descendants = _useTabs.descendants,
      ctx = _objectWithoutPropertiesLoose(_useTabs, _excluded2);

  var context = React__namespace.useMemo(function () {
    return ctx;
  }, [ctx]);
  var rootProps = utils.omit(htmlProps, ["isFitted"]);
  return /*#__PURE__*/React__namespace.createElement(TabsDescendantsProvider, {
    value: descendants
  }, /*#__PURE__*/React__namespace.createElement(TabsProvider, {
    value: context
  }, /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    className: utils.cx("chakra-tabs", className),
    ref: ref
  }, rootProps, {
    __css: styles.root
  }), children))));
});

if (utils.__DEV__) {
  Tabs.displayName = "Tabs";
}

/**
 * Tab button used to activate a specific tab panel. It renders a `button`,
 * and is responsible for automatic and manual selection modes.
 */
var Tab = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useStyles();
  var tabProps = useTab(_extends({}, props, {
    ref: ref
  }));

  var tabStyles = _extends({
    outline: "0",
    display: "flex",
    alignItems: "center",
    justifyContent: "center"
  }, styles.tab);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.button, _extends({}, tabProps, {
    className: utils.cx("chakra-tabs__tab", props.className),
    __css: tabStyles
  }));
});

if (utils.__DEV__) {
  Tab.displayName = "Tab";
}

/**
 * TabList is used to manage a list of tab buttons. It renders a `div` by default,
 * and is responsible the keyboard interaction between tabs.
 */
var TabList = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var tablistProps = useTabList(_extends({}, props, {
    ref: ref
  }));
  var styles = system.useStyles();

  var tablistStyles = _extends({
    display: "flex"
  }, styles.tablist);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, tablistProps, {
    className: utils.cx("chakra-tabs__tablist", props.className),
    __css: tablistStyles
  }));
});

if (utils.__DEV__) {
  TabList.displayName = "TabList";
}

/**
 * TabPanel
 * Used to render the content for a specific tab.
 */
var TabPanel = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var panelProps = useTabPanel(_extends({}, props, {
    ref: ref
  }));
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    outline: "0"
  }, panelProps, {
    className: utils.cx("chakra-tabs__tab-panel", props.className),
    __css: styles.tabpanel
  }));
});

if (utils.__DEV__) {
  TabPanel.displayName = "TabPanel";
}

/**
 * TabPanel
 *
 * Used to manage the rendering of multiple tab panels. It uses
 * `cloneElement` to hide/show tab panels.
 *
 * It renders a `div` by default.
 */
var TabPanels = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var panelsProps = useTabPanels(props);
  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({}, panelsProps, {
    width: "100%",
    ref: ref,
    className: utils.cx("chakra-tabs__tab-panels", props.className),
    __css: styles.tabpanels
  }));
});

if (utils.__DEV__) {
  TabPanels.displayName = "TabPanels";
}

/**
 * TabIndicator
 *
 * Used to render an active tab indicator that animates between
 * selected tabs.
 */
var TabIndicator = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var indicatorStyle = useTabIndicator();

  var style = _extends({}, props.style, indicatorStyle);

  var styles = system.useStyles();
  return /*#__PURE__*/React__namespace.createElement(system.chakra.div, _extends({
    ref: ref
  }, props, {
    className: utils.cx("chakra-tabs__tab-indicator", props.className),
    style: style,
    __css: styles.indicator
  }));
});

if (utils.__DEV__) {
  TabIndicator.displayName = "TabIndicator";
}

exports.Tab = Tab;
exports.TabIndicator = TabIndicator;
exports.TabList = TabList;
exports.TabPanel = TabPanel;
exports.TabPanels = TabPanels;
exports.Tabs = Tabs;
exports.TabsDescendantsProvider = TabsDescendantsProvider;
exports.TabsProvider = TabsProvider;
exports.useTab = useTab;
exports.useTabIndicator = useTabIndicator;
exports.useTabList = useTabList;
exports.useTabPanel = useTabPanel;
exports.useTabPanels = useTabPanels;
exports.useTabs = useTabs;
exports.useTabsContext = useTabsContext;
exports.useTabsDescendant = useTabsDescendant;
exports.useTabsDescendants = useTabsDescendants;
exports.useTabsDescendantsContext = useTabsDescendantsContext;
