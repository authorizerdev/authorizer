import { correctNodes } from './utils/correctFocus';
import { pickFocusable } from './utils/firstFocus';
import { isGuard } from './utils/is';
export var NEW_FOCUS = 'NEW_FOCUS';
/**
 * Main solver for the "find next focus" question
 * @param innerNodes - used to control "return focus"
 * @param innerTabbables - used to control "autofocus"
 * @param outerNodes
 * @param activeElement
 * @param lastNode
 * @returns {number|string|undefined|*}
 */
export var newFocus = function (innerNodes, innerTabbables, outerNodes, activeElement, lastNode) {
    var cnt = innerNodes.length;
    var firstFocus = innerNodes[0];
    var lastFocus = innerNodes[cnt - 1];
    var isOnGuard = isGuard(activeElement);
    // focus is inside
    if (activeElement && innerNodes.indexOf(activeElement) >= 0) {
        return undefined;
    }
    var activeIndex = activeElement !== undefined ? outerNodes.indexOf(activeElement) : -1;
    var lastIndex = lastNode ? outerNodes.indexOf(lastNode) : activeIndex;
    var lastNodeInside = lastNode ? innerNodes.indexOf(lastNode) : -1;
    // no active focus (or focus is on the body)
    if (activeIndex === -1) {
        // known fallback
        if (lastNodeInside !== -1) {
            return lastNodeInside;
        }
        return NEW_FOCUS;
    }
    // new focus, nothing to calculate
    if (lastNodeInside === -1) {
        return NEW_FOCUS;
    }
    var indexDiff = activeIndex - lastIndex;
    var firstNodeIndex = outerNodes.indexOf(firstFocus);
    var lastNodeIndex = outerNodes.indexOf(lastFocus);
    var correctedNodes = correctNodes(outerNodes);
    var currentFocusableIndex = activeElement !== undefined ? correctedNodes.indexOf(activeElement) : -1;
    var previousFocusableIndex = lastNode ? correctedNodes.indexOf(lastNode) : currentFocusableIndex;
    var tabbableNodes = correctedNodes.filter(function (node) { return node.tabIndex >= 0; });
    var currentTabbableIndex = activeElement !== undefined ? tabbableNodes.indexOf(activeElement) : -1;
    var previousTabbableIndex = lastNode ? tabbableNodes.indexOf(lastNode) : currentTabbableIndex;
    var focusIndexDiff = currentTabbableIndex >= 0 && previousTabbableIndex >= 0
        ? // old/new are tabbables, measure distance in tabbable space
            previousTabbableIndex - currentTabbableIndex
        : // or else measure in focusable space
            previousFocusableIndex - currentFocusableIndex;
    // old focus
    if (!indexDiff && lastNodeInside >= 0) {
        return lastNodeInside;
    }
    // no tabbable elements, autofocus is not possible
    if (innerTabbables.length === 0) {
        // an edge case with no tabbable elements
        // return the last focusable one
        // with some probability this will prevent focus from cycling across the lock, but there is no tabbale elements to cycle to
        return lastNodeInside;
    }
    var returnFirstNode = pickFocusable(innerNodes, innerTabbables[0]);
    var returnLastNode = pickFocusable(innerNodes, innerTabbables[innerTabbables.length - 1]);
    // first element
    if (activeIndex <= firstNodeIndex && isOnGuard && Math.abs(indexDiff) > 1) {
        return returnLastNode;
    }
    // last element
    if (activeIndex >= lastNodeIndex && isOnGuard && Math.abs(indexDiff) > 1) {
        return returnFirstNode;
    }
    // jump out, but not on the guard
    if (indexDiff && Math.abs(focusIndexDiff) > 1) {
        return lastNodeInside;
    }
    // focus above lock
    if (activeIndex <= firstNodeIndex) {
        return returnLastNode;
    }
    // focus below lock
    if (activeIndex > lastNodeIndex) {
        return returnFirstNode;
    }
    // index is inside tab order, but outside Lock
    if (indexDiff) {
        if (Math.abs(indexDiff) > 1) {
            return lastNodeInside;
        }
        return (cnt + lastNodeInside + indexDiff) % cnt;
    }
    // do nothing
    return undefined;
};
