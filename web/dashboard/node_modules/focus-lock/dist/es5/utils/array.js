"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toArray = function (a) {
    var ret = Array(a.length);
    for (var i = 0; i < a.length; ++i) {
        ret[i] = a[i];
    }
    return ret;
};
exports.asArray = function (a) { return (Array.isArray(a) ? a : [a]); };
