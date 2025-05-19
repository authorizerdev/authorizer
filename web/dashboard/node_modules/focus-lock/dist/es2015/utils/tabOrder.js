import { toArray } from './array';
export var tabSort = function (a, b) {
    var aTab = Math.max(0, a.tabIndex);
    var bTab = Math.max(0, b.tabIndex);
    var tabDiff = aTab - bTab;
    var indexDiff = a.index - b.index;
    if (tabDiff) {
        if (!aTab) {
            return 1;
        }
        if (!bTab) {
            return -1;
        }
    }
    return tabDiff || indexDiff;
};
var getTabIndex = function (node) {
    if (node.tabIndex < 0) {
        // all "focusable" elements are already preselected
        // but some might have implicit negative tabIndex
        // return 0 for <audio without tabIndex attribute - it is "tabbable"
        if (!node.hasAttribute('tabindex')) {
            return 0;
        }
    }
    return node.tabIndex;
};
export var orderByTabIndex = function (nodes, filterNegative, keepGuards) {
    return toArray(nodes)
        .map(function (node, index) {
        var tabIndex = getTabIndex(node);
        return {
            node: node,
            index: index,
            tabIndex: keepGuards && tabIndex === -1 ? ((node.dataset || {}).focusGuard ? 0 : -1) : tabIndex,
        };
    })
        .filter(function (data) { return !filterNegative || data.tabIndex >= 0; })
        .sort(tabSort);
};
