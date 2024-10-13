'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
var React = require('react');
var hooks = require('@chakra-ui/hooks');
var reactDom = require('react-dom');

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

var _createContext$1 = reactUtils.createContext({
  strict: false,
  name: "PortalManagerContext"
}),
    PortalManagerContextProvider = _createContext$1[0],
    usePortalManager = _createContext$1[1];
function PortalManager(props) {
  var children = props.children,
      zIndex = props.zIndex;
  return /*#__PURE__*/React__namespace.createElement(PortalManagerContextProvider, {
    value: {
      zIndex: zIndex
    }
  }, children);
}

if (utils.__DEV__) {
  PortalManager.displayName = "PortalManager";
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

var _excluded = ["containerRef"];

var _createContext = reactUtils.createContext({
  strict: false,
  name: "PortalContext"
}),
    PortalContextProvider = _createContext[0],
    usePortalContext = _createContext[1];

var PORTAL_CLASSNAME = "chakra-portal";
var PORTAL_SELECTOR = ".chakra-portal";

var Container = function Container(props) {
  return /*#__PURE__*/React__namespace.createElement("div", {
    className: "chakra-portal-zIndex",
    style: {
      position: "absolute",
      zIndex: props.zIndex,
      top: 0,
      left: 0,
      right: 0 // NB: Don't add `bottom: 0`, it makes the entire app unusable
      // @see https://github.com/chakra-ui/chakra-ui/issues/3201

    }
  }, props.children);
};
/**
 * Portal that uses `document.body` as container
 */


var DefaultPortal = function DefaultPortal(props) {
  var appendToParentPortal = props.appendToParentPortal,
      children = props.children;
  var tempNode = React__namespace.useRef(null);
  var portal = React__namespace.useRef(null);
  var forceUpdate = hooks.useForceUpdate();
  var parentPortal = usePortalContext();
  var manager = usePortalManager();
  hooks.useSafeLayoutEffect(function () {
    if (!tempNode.current) return;
    var doc = tempNode.current.ownerDocument;
    var host = appendToParentPortal ? parentPortal != null ? parentPortal : doc.body : doc.body;
    if (!host) return;
    portal.current = doc.createElement("div");
    portal.current.className = PORTAL_CLASSNAME;
    host.appendChild(portal.current);
    forceUpdate();
    var portalNode = portal.current;
    return function () {
      if (host.contains(portalNode)) {
        host.removeChild(portalNode);
      }
    };
  }, []);

  var _children = manager != null && manager.zIndex ? /*#__PURE__*/React__namespace.createElement(Container, {
    zIndex: manager == null ? void 0 : manager.zIndex
  }, children) : children;

  return portal.current ? /*#__PURE__*/reactDom.createPortal( /*#__PURE__*/React__namespace.createElement(PortalContextProvider, {
    value: portal.current
  }, _children), portal.current) : /*#__PURE__*/React__namespace.createElement("span", {
    ref: tempNode
  });
};

/**
 * Portal that uses a custom container
 */
var ContainerPortal = function ContainerPortal(props) {
  var children = props.children,
      containerRef = props.containerRef,
      appendToParentPortal = props.appendToParentPortal;
  var containerEl = containerRef.current;
  var host = containerEl != null ? containerEl : utils.isBrowser ? document.body : undefined;
  var portal = React__namespace.useMemo(function () {
    var node = containerEl == null ? void 0 : containerEl.ownerDocument.createElement("div");
    if (node) node.className = PORTAL_CLASSNAME;
    return node;
  }, [containerEl]);
  var forceUpdate = hooks.useForceUpdate();
  hooks.useSafeLayoutEffect(function () {
    forceUpdate();
  }, []);
  hooks.useSafeLayoutEffect(function () {
    if (!portal || !host) return;
    host.appendChild(portal);
    return function () {
      host.removeChild(portal);
    };
  }, [portal, host]);

  if (host && portal) {
    return /*#__PURE__*/reactDom.createPortal( /*#__PURE__*/React__namespace.createElement(PortalContextProvider, {
      value: appendToParentPortal ? portal : null
    }, children), portal);
  }

  return null;
};

/**
 * Portal
 *
 * Declarative component used to render children into a DOM node
 * that exists outside the DOM hierarchy of the parent component.
 *
 * @see Docs https://chakra-ui.com/portal
 */
function Portal(props) {
  var containerRef = props.containerRef,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  return containerRef ? /*#__PURE__*/React__namespace.createElement(ContainerPortal, _extends({
    containerRef: containerRef
  }, rest)) : /*#__PURE__*/React__namespace.createElement(DefaultPortal, rest);
}
Portal.defaultProps = {
  appendToParentPortal: true
};
Portal.className = PORTAL_CLASSNAME;
Portal.selector = PORTAL_SELECTOR;

if (utils.__DEV__) {
  Portal.displayName = "Portal";
}

exports.Portal = Portal;
exports.PortalManager = PortalManager;
exports.usePortalManager = usePortalManager;
