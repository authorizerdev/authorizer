"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.pickFocusable = exports.pickFirstFocus = void 0;
var correctFocus_1 = require("./correctFocus");
var pickFirstFocus = function (nodes) {
    if (nodes[0] && nodes.length > 1) {
        return (0, correctFocus_1.correctNode)(nodes[0], nodes);
    }
    return nodes[0];
};
exports.pickFirstFocus = pickFirstFocus;
var pickFocusable = function (nodes, node) {
    return nodes.indexOf((0, correctFocus_1.correctNode)(node, nodes));
};
exports.pickFocusable = pickFocusable;
