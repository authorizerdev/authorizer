import { getAllAffectedNodes } from './utils/all-affected';
import { isGuard, isNotAGuard } from './utils/is';
import { getTopCommonParent } from './utils/parenting';
import { orderByTabIndex } from './utils/tabOrder';
import { getFocusables } from './utils/tabUtils';
/**
 * traverses all related nodes (including groups) returning a list of all nodes(outer and internal) with meta information
 * This is low-level API!
 * @returns list of focusable elements inside a given top(!) node.
 * @see {@link getFocusableNodes} providing a simpler API
 */
export var expandFocusableNodes = function (topNode) {
    var entries = getAllAffectedNodes(topNode).filter(isNotAGuard);
    var commonParent = getTopCommonParent(topNode, topNode, entries);
    var outerNodes = orderByTabIndex(getFocusables([commonParent], true), true, true);
    var innerElements = getFocusables(entries, false);
    return outerNodes.map(function (_a) {
        var node = _a.node, index = _a.index;
        return ({
            node: node,
            index: index,
            lockItem: innerElements.indexOf(node) >= 0,
            guard: isGuard(node),
        });
    });
};
