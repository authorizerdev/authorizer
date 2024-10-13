"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var constants_1 = require("./constants");
var array_1 = require("./utils/array");
exports.focusIsHidden = function () {
    return document &&
        array_1.toArray(document.querySelectorAll("[" + constants_1.FOCUS_ALLOW + "]")).some(function (node) { return node.contains(document.activeElement); });
};
