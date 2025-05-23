"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.expandFocusableNodes = void 0;
var all_affected_1 = require("./utils/all-affected");
var is_1 = require("./utils/is");
var parenting_1 = require("./utils/parenting");
var tabOrder_1 = require("./utils/tabOrder");
var tabUtils_1 = require("./utils/tabUtils");
/**
 * traverses all related nodes (including groups) returning a list of all nodes(outer and internal) with meta information
 * This is low-level API!
 * @returns list of focusable elements inside a given top(!) node.
 * @see {@link getFocusableNodes} providing a simpler API
 */
var expandFocusableNodes = function (topNode) {
    var entries = (0, all_affected_1.getAllAffectedNodes)(topNode).filter(is_1.isNotAGuard);
    var commonParent = (0, parenting_1.getTopCommonParent)(topNode, topNode, entries);
    var outerNodes = (0, tabOrder_1.orderByTabIndex)((0, tabUtils_1.getFocusables)([commonParent], true), true, true);
    var innerElements = (0, tabUtils_1.getFocusables)(entries, false);
    return outerNodes.map(function (_a) {
        var node = _a.node, index = _a.index;
        return ({
            node: node,
            index: index,
            lockItem: innerElements.indexOf(node) >= 0,
            guard: (0, is_1.isGuard)(node),
        });
    });
};
exports.expandFocusableNodes = expandFocusableNodes;
