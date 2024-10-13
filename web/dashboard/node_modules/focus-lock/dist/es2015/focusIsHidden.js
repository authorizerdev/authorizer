import { FOCUS_ALLOW } from './constants';
import { toArray } from './utils/array';
export var focusIsHidden = function () {
    return document &&
        toArray(document.querySelectorAll("[" + FOCUS_ALLOW + "]")).some(function (node) { return node.contains(document.activeElement); });
};
