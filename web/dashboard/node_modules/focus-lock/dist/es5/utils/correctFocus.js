"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var isRadio = function (node) { return node.tagName === 'INPUT' && node.type === 'radio'; };
var findSelectedRadio = function (node, nodes) {
    return nodes
        .filter(isRadio)
        .filter(function (el) { return el.name === node.name; })
        .filter(function (el) { return el.checked; })[0] || node;
};
exports.correctNode = function (node, nodes) {
    if (isRadio(node) && node.name) {
        return findSelectedRadio(node, nodes);
    }
    return node;
};
exports.correctNodes = function (nodes) {
    var resultSet = new Set();
    nodes.forEach(function (node) { return resultSet.add(exports.correctNode(node, nodes)); });
    return nodes.filter(function (node) { return resultSet.has(node); });
};
