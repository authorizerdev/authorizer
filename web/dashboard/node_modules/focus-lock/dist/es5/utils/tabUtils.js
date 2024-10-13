"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var constants_1 = require("../constants");
var array_1 = require("./array");
var tabbables_1 = require("./tabbables");
var queryTabbables = tabbables_1.tabbables.join(',');
var queryGuardTabbables = queryTabbables + ", [data-focus-guard]";
exports.getFocusables = function (parents, withGuards) {
    return parents.reduce(function (acc, parent) {
        return acc.concat(array_1.toArray(parent.querySelectorAll(withGuards ? queryGuardTabbables : queryTabbables)), parent.parentNode
            ? array_1.toArray(parent.parentNode.querySelectorAll(queryTabbables)).filter(function (node) { return node === parent; })
            : []);
    }, []);
};
exports.getParentAutofocusables = function (parent) {
    var parentFocus = parent.querySelectorAll("[" + constants_1.FOCUS_AUTO + "]");
    return array_1.toArray(parentFocus)
        .map(function (node) { return exports.getFocusables([node]); })
        .reduce(function (acc, nodes) { return acc.concat(nodes); }, []);
};
