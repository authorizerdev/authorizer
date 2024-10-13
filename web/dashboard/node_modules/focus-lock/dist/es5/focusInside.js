"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var all_affected_1 = require("./utils/all-affected");
var array_1 = require("./utils/array");
var focusInFrame = function (frame) { return frame === document.activeElement; };
var focusInsideIframe = function (topNode) {
    return Boolean(array_1.toArray(topNode.querySelectorAll('iframe')).some(function (node) { return focusInFrame(node); }));
};
exports.focusInside = function (topNode) {
    var activeElement = document && document.activeElement;
    if (!activeElement || (activeElement.dataset && activeElement.dataset.focusGuard)) {
        return false;
    }
    return all_affected_1.getAllAffectedNodes(topNode).reduce(function (result, node) { return result || node.contains(activeElement) || focusInsideIframe(node); }, false);
};
