import { toArray } from './array';
import { isVisibleCached, notHiddenInput } from './is';
import { orderByTabIndex } from './tabOrder';
import { getFocusables, getParentAutofocusables } from './tabUtils';
export var filterFocusable = function (nodes, visibilityCache) {
    return toArray(nodes)
        .filter(function (node) { return isVisibleCached(visibilityCache, node); })
        .filter(function (node) { return notHiddenInput(node); });
};
export var getTabbableNodes = function (topNodes, visibilityCache, withGuards) {
    return orderByTabIndex(filterFocusable(getFocusables(topNodes, withGuards), visibilityCache), true, withGuards);
};
export var getAllTabbableNodes = function (topNodes, visibilityCache) {
    return orderByTabIndex(filterFocusable(getFocusables(topNodes), visibilityCache), false);
};
export var parentAutofocusables = function (topNode, visibilityCache) {
    return filterFocusable(getParentAutofocusables(topNode), visibilityCache);
};
