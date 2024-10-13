"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var constants_1 = require("../constants");
var array_1 = require("./array");
var filterNested = function (nodes) {
    var contained = new Set();
    var l = nodes.length;
    for (var i = 0; i < l; i += 1) {
        for (var j = i + 1; j < l; j += 1) {
            var position = nodes[i].compareDocumentPosition(nodes[j]);
            if ((position & Node.DOCUMENT_POSITION_CONTAINED_BY) > 0) {
                contained.add(j);
            }
            if ((position & Node.DOCUMENT_POSITION_CONTAINS) > 0) {
                contained.add(i);
            }
        }
    }
    return nodes.filter(function (_, index) { return !contained.has(index); });
};
var getTopParent = function (node) {
    return node.parentNode ? getTopParent(node.parentNode) : node;
};
exports.getAllAffectedNodes = function (node) {
    var nodes = array_1.asArray(node);
    return nodes.filter(Boolean).reduce(function (acc, currentNode) {
        var group = currentNode.getAttribute(constants_1.FOCUS_GROUP);
        acc.push.apply(acc, (group
            ? filterNested(array_1.toArray(getTopParent(currentNode).querySelectorAll("[" + constants_1.FOCUS_GROUP + "=\"" + group + "\"]:not([" + constants_1.FOCUS_DISABLED + "=\"disabled\"])")))
            : [currentNode]));
        return acc;
    }, []);
};
