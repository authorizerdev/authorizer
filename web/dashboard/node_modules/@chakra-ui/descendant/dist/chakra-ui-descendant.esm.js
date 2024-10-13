import { createContext, mergeRefs } from '@chakra-ui/react-utils';
import { useLayoutEffect, useEffect, useRef, useState } from 'react';

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

/**
 * Sort an array of DOM nodes according to the HTML tree order
 * @see http://www.w3.org/TR/html5/infrastructure.html#tree-order
 */

function sortNodes(nodes) {
  return nodes.sort(function (a, b) {
    var compare = a.compareDocumentPosition(b);

    if (compare & Node.DOCUMENT_POSITION_FOLLOWING || compare & Node.DOCUMENT_POSITION_CONTAINED_BY) {
      // a < b
      return -1;
    }

    if (compare & Node.DOCUMENT_POSITION_PRECEDING || compare & Node.DOCUMENT_POSITION_CONTAINS) {
      // a > b
      return 1;
    }

    if (compare & Node.DOCUMENT_POSITION_DISCONNECTED || compare & Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC) {
      throw Error("Cannot sort the given nodes.");
    } else {
      return 0;
    }
  });
}
var isElement = function isElement(el) {
  return typeof el == "object" && "nodeType" in el && el.nodeType === Node.ELEMENT_NODE;
};
function getNextIndex(current, max, loop) {
  var next = current + 1;
  if (loop && next >= max) next = 0;
  return next;
}
function getPrevIndex(current, max, loop) {
  var next = current - 1;
  if (loop && next < 0) next = max;
  return next;
}
var useSafeLayoutEffect = typeof window !== "undefined" ? useLayoutEffect : useEffect;
var cast = function cast(value) {
  return value;
};

/**
 * @internal
 *
 * Class to manage descendants and their relative indices in the DOM.
 * It uses `node.compareDocumentPosition(...)` under the hood
 */
var DescendantsManager = function DescendantsManager() {
  var _this = this;

  this.descendants = new Map();

  this.register = function (nodeOrOptions) {
    if (nodeOrOptions == null) return;

    if (isElement(nodeOrOptions)) {
      return _this.registerNode(nodeOrOptions);
    }

    return function (node) {
      _this.registerNode(node, nodeOrOptions);
    };
  };

  this.unregister = function (node) {
    _this.descendants["delete"](node);

    var sorted = sortNodes(Array.from(_this.descendants.keys()));

    _this.assignIndex(sorted);
  };

  this.destroy = function () {
    _this.descendants.clear();
  };

  this.assignIndex = function (descendants) {
    _this.descendants.forEach(function (descendant) {
      var index = descendants.indexOf(descendant.node);
      descendant.index = index;
      descendant.node.dataset.index = descendant.index.toString();
    });
  };

  this.count = function () {
    return _this.descendants.size;
  };

  this.enabledCount = function () {
    return _this.enabledValues().length;
  };

  this.values = function () {
    var values = Array.from(_this.descendants.values());
    return values.sort(function (a, b) {
      return a.index - b.index;
    });
  };

  this.enabledValues = function () {
    return _this.values().filter(function (descendant) {
      return !descendant.disabled;
    });
  };

  this.item = function (index) {
    if (_this.count() === 0) return undefined;
    return _this.values()[index];
  };

  this.enabledItem = function (index) {
    if (_this.enabledCount() === 0) return undefined;
    return _this.enabledValues()[index];
  };

  this.first = function () {
    return _this.item(0);
  };

  this.firstEnabled = function () {
    return _this.enabledItem(0);
  };

  this.last = function () {
    return _this.item(_this.descendants.size - 1);
  };

  this.lastEnabled = function () {
    var lastIndex = _this.enabledValues().length - 1;
    return _this.enabledItem(lastIndex);
  };

  this.indexOf = function (node) {
    var _this$descendants$get, _this$descendants$get2;

    if (!node) return -1;
    return (_this$descendants$get = (_this$descendants$get2 = _this.descendants.get(node)) == null ? void 0 : _this$descendants$get2.index) != null ? _this$descendants$get : -1;
  };

  this.enabledIndexOf = function (node) {
    if (node == null) return -1;
    return _this.enabledValues().findIndex(function (i) {
      return i.node.isSameNode(node);
    });
  };

  this.next = function (index, loop) {
    if (loop === void 0) {
      loop = true;
    }

    var next = getNextIndex(index, _this.count(), loop);
    return _this.item(next);
  };

  this.nextEnabled = function (index, loop) {
    if (loop === void 0) {
      loop = true;
    }

    var item = _this.item(index);

    if (!item) return;

    var enabledIndex = _this.enabledIndexOf(item.node);

    var nextEnabledIndex = getNextIndex(enabledIndex, _this.enabledCount(), loop);
    return _this.enabledItem(nextEnabledIndex);
  };

  this.prev = function (index, loop) {
    if (loop === void 0) {
      loop = true;
    }

    var prev = getPrevIndex(index, _this.count() - 1, loop);
    return _this.item(prev);
  };

  this.prevEnabled = function (index, loop) {
    if (loop === void 0) {
      loop = true;
    }

    var item = _this.item(index);

    if (!item) return;

    var enabledIndex = _this.enabledIndexOf(item.node);

    var prevEnabledIndex = getPrevIndex(enabledIndex, _this.enabledCount() - 1, loop);
    return _this.enabledItem(prevEnabledIndex);
  };

  this.registerNode = function (node, options) {
    if (!node || _this.descendants.has(node)) return;
    var keys = Array.from(_this.descendants.keys()).concat(node);
    var sorted = sortNodes(keys);

    if (options != null && options.disabled) {
      options.disabled = !!options.disabled;
    }

    var descendant = _extends({
      node: node,
      index: -1
    }, options);

    _this.descendants.set(node, descendant);

    _this.assignIndex(sorted);
  };
};

/**
 * @internal
 * React hook that initializes the DescendantsManager
 */

function useDescendants() {
  var descendants = useRef(new DescendantsManager());
  useSafeLayoutEffect(function () {
    return function () {
      return descendants.current.destroy();
    };
  });
  return descendants.current;
}

/* -------------------------------------------------------------------------------------------------
 * Descendants context to be used in component-land.
  - Mount the `DescendantsContextProvider` at the root of the component
  - Call `useDescendantsContext` anywhere you need access to the descendants information

  NB:  I recommend using `createDescendantContext` below
 * -----------------------------------------------------------------------------------------------*/
var _createContext = createContext({
  name: "DescendantsProvider",
  errorMessage: "useDescendantsContext must be used within DescendantsProvider"
}),
    DescendantsContextProvider = _createContext[0],
    useDescendantsContext = _createContext[1];
/**
 * @internal
 * This hook provides information a descendant such as:
 * - Its index compared to other descendants
 * - ref callback to register the descendant
 * - Its enabled index compared to other enabled descendants
 */


function useDescendant(options) {
  var descendants = useDescendantsContext();

  var _useState = useState(-1),
      index = _useState[0],
      setIndex = _useState[1];

  var ref = useRef(null);
  useSafeLayoutEffect(function () {
    return function () {
      if (!ref.current) return;
      descendants.unregister(ref.current);
    };
  }, []);
  useSafeLayoutEffect(function () {
    if (!ref.current) return;
    var dataIndex = Number(ref.current.dataset.index);

    if (index != dataIndex && !Number.isNaN(dataIndex)) {
      setIndex(dataIndex);
    }
  });
  var refCallback = options ? cast(descendants.register(options)) : cast(descendants.register);
  return {
    descendants: descendants,
    index: index,
    enabledIndex: descendants.enabledIndexOf(ref.current),
    register: mergeRefs(refCallback, ref)
  };
}
/* -------------------------------------------------------------------------------------------------
 * Function that provides strongly typed versions of the context provider and hooks above.
   To be used in component-land
 * -----------------------------------------------------------------------------------------------*/


function createDescendantContext() {
  var ContextProvider = cast(DescendantsContextProvider);

  var _useDescendantsContext = function _useDescendantsContext() {
    return cast(useDescendantsContext());
  };

  var _useDescendant = function _useDescendant(options) {
    return useDescendant(options);
  };

  var _useDescendants = function _useDescendants() {
    return useDescendants();
  };

  return [// context provider
  ContextProvider, // call this when you need to read from context
  _useDescendantsContext, // descendants state information, to be called and passed to `ContextProvider`
  _useDescendants, // descendant index information
  _useDescendant];
}

export { createDescendantContext, createDescendantContext as default };
