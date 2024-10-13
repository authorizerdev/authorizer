import { getAllAffectedNodes } from './utils/all-affected';
import { toArray } from './utils/array';
var focusInFrame = function (frame) { return frame === document.activeElement; };
var focusInsideIframe = function (topNode) {
    return Boolean(toArray(topNode.querySelectorAll('iframe')).some(function (node) { return focusInFrame(node); }));
};
export var focusInside = function (topNode) {
    var activeElement = document && document.activeElement;
    if (!activeElement || (activeElement.dataset && activeElement.dataset.focusGuard)) {
        return false;
    }
    return getAllAffectedNodes(topNode).reduce(function (result, node) { return result || node.contains(activeElement) || focusInsideIframe(node); }, false);
};
