"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var correctFocus_1 = require("./utils/correctFocus");
var firstFocus_1 = require("./utils/firstFocus");
var is_1 = require("./utils/is");
exports.NEW_FOCUS = 'NEW_FOCUS';
exports.newFocus = function (innerNodes, outerNodes, activeElement, lastNode) {
    var cnt = innerNodes.length;
    var firstFocus = innerNodes[0];
    var lastFocus = innerNodes[cnt - 1];
    var isOnGuard = is_1.isGuard(activeElement);
    if (innerNodes.indexOf(activeElement) >= 0) {
        return undefined;
    }
    var activeIndex = outerNodes.indexOf(activeElement);
    var lastIndex = lastNode ? outerNodes.indexOf(lastNode) : activeIndex;
    var lastNodeInside = lastNode ? innerNodes.indexOf(lastNode) : -1;
    var indexDiff = activeIndex - lastIndex;
    var firstNodeIndex = outerNodes.indexOf(firstFocus);
    var lastNodeIndex = outerNodes.indexOf(lastFocus);
    var correctedNodes = correctFocus_1.correctNodes(outerNodes);
    var correctedIndexDiff = correctedNodes.indexOf(activeElement) - (lastNode ? correctedNodes.indexOf(lastNode) : activeIndex);
    var returnFirstNode = firstFocus_1.pickFocusable(innerNodes, 0);
    var returnLastNode = firstFocus_1.pickFocusable(innerNodes, cnt - 1);
    if (activeIndex === -1 || lastNodeInside === -1) {
        return exports.NEW_FOCUS;
    }
    if (!indexDiff && lastNodeInside >= 0) {
        return lastNodeInside;
    }
    if (activeIndex <= firstNodeIndex && isOnGuard && Math.abs(indexDiff) > 1) {
        return returnLastNode;
    }
    if (activeIndex >= lastNodeIndex && isOnGuard && Math.abs(indexDiff) > 1) {
        return returnFirstNode;
    }
    if (indexDiff && Math.abs(correctedIndexDiff) > 1) {
        return lastNodeInside;
    }
    if (activeIndex <= firstNodeIndex) {
        return returnLastNode;
    }
    if (activeIndex > lastNodeIndex) {
        return returnFirstNode;
    }
    if (indexDiff) {
        if (Math.abs(indexDiff) > 1) {
            return lastNodeInside;
        }
        return (cnt + lastNodeInside + indexDiff) % cnt;
    }
    return undefined;
};
