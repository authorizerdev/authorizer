import { correctNode } from './correctFocus';
export var pickFirstFocus = function (nodes) {
    if (nodes[0] && nodes.length > 1) {
        return correctNode(nodes[0], nodes);
    }
    return nodes[0];
};
export var pickFocusable = function (nodes, node) {
    return nodes.indexOf(correctNode(node, nodes));
};
