"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var focusMerge_1 = require("./focusMerge");
exports.focusOn = function (target) {
    target.focus();
    if ('contentWindow' in target && target.contentWindow) {
        target.contentWindow.focus();
    }
};
var guardCount = 0;
var lockDisabled = false;
exports.setFocus = function (topNode, lastNode) {
    var focusable = focusMerge_1.getFocusMerge(topNode, lastNode);
    if (lockDisabled) {
        return;
    }
    if (focusable) {
        if (guardCount > 2) {
            console.error('FocusLock: focus-fighting detected. Only one focus management system could be active. ' +
                'See https://github.com/theKashey/focus-lock/#focus-fighting');
            lockDisabled = true;
            setTimeout(function () {
                lockDisabled = false;
            }, 1);
            return;
        }
        guardCount++;
        exports.focusOn(focusable.node);
        guardCount--;
    }
};
