"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var correctFocus_1 = require("./correctFocus");
exports.pickFirstFocus = function (nodes) {
    if (nodes[0] && nodes.length > 1) {
        return correctFocus_1.correctNode(nodes[0], nodes);
    }
    return nodes[0];
};
exports.pickFocusable = function (nodes, index) {
    if (nodes.length > 1) {
        return nodes.indexOf(correctFocus_1.correctNode(nodes[index], nodes));
    }
    return index;
};
