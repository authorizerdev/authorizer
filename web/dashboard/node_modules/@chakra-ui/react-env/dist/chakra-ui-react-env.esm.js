import { __DEV__, isBrowser } from '@chakra-ui/utils';
import React, { useContext, useState, useMemo, createContext } from 'react';

var doc = {
  body: {
    classList: {
      add: function add() {},
      remove: function remove() {}
    }
  },
  addEventListener: function addEventListener() {},
  removeEventListener: function removeEventListener() {},
  activeElement: {
    blur: function blur() {},
    nodeName: ""
  },
  querySelector: function querySelector() {
    return null;
  },
  querySelectorAll: function querySelectorAll() {
    return [];
  },
  getElementById: function getElementById() {
    return null;
  },
  createEvent: function createEvent() {
    return {
      initEvent: function initEvent() {}
    };
  },
  createElement: function createElement() {
    return {
      children: [],
      childNodes: [],
      style: {},
      setAttribute: function setAttribute() {},
      getElementsByTagName: function getElementsByTagName() {
        return [];
      }
    };
  }
};
var ssrDocument = doc;

var noop = function noop() {};

var win = {
  document: ssrDocument,
  navigator: {
    userAgent: ""
  },
  CustomEvent: function CustomEvent() {
    return this;
  },
  addEventListener: noop,
  removeEventListener: noop,
  getComputedStyle: function getComputedStyle() {
    return {
      getPropertyValue: function getPropertyValue() {
        return "";
      }
    };
  },
  matchMedia: function matchMedia() {
    return {
      matches: false,
      addListener: noop,
      removeListener: noop
    };
  },
  requestAnimationFrame: function requestAnimationFrame(callback) {
    if (typeof setTimeout === "undefined") {
      callback();
      return null;
    }

    return setTimeout(callback, 0);
  },
  cancelAnimationFrame: function cancelAnimationFrame(id) {
    if (typeof setTimeout === "undefined") return;
    clearTimeout(id);
  },
  setTimeout: function setTimeout() {
    return 0;
  },
  clearTimeout: noop,
  setInterval: function setInterval() {
    return 0;
  },
  clearInterval: noop
};
var ssrWindow = win;

var mockEnv = {
  window: ssrWindow,
  document: ssrDocument
};
var defaultEnv = isBrowser ? {
  window: window,
  document: document
} : mockEnv;
var EnvironmentContext = /*#__PURE__*/createContext(defaultEnv);

if (__DEV__) {
  EnvironmentContext.displayName = "EnvironmentContext";
}

function useEnvironment() {
  return useContext(EnvironmentContext);
}
function EnvironmentProvider(props) {
  var children = props.children,
      environmentProp = props.environment;

  var _useState = useState(null),
      node = _useState[0],
      setNode = _useState[1];

  var context = useMemo(function () {
    var _ref;

    var doc = node == null ? void 0 : node.ownerDocument;
    var win = node == null ? void 0 : node.ownerDocument.defaultView;
    var nodeEnv = doc ? {
      document: doc,
      window: win
    } : undefined;
    var env = (_ref = environmentProp != null ? environmentProp : nodeEnv) != null ? _ref : defaultEnv;
    return env;
  }, [node, environmentProp]);
  var showEnvGetter = !node && !environmentProp;
  return /*#__PURE__*/React.createElement(EnvironmentContext.Provider, {
    value: context
  }, children, showEnvGetter && /*#__PURE__*/React.createElement("span", {
    ref: function ref(el) {
      if (el) setNode(el);
    }
  }));
}

if (__DEV__) {
  EnvironmentProvider.displayName = "EnvironmentProvider";
}

export { EnvironmentProvider, useEnvironment };
