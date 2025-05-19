import { focusMerge } from './focusMerge';
export var focusOn = function (target, focusOptions) {
    if ('focus' in target) {
        target.focus(focusOptions);
    }
    if ('contentWindow' in target && target.contentWindow) {
        target.contentWindow.focus();
    }
};
var guardCount = 0;
var lockDisabled = false;
/**
 * Control focus at a given node.
 * The last focused element will help to determine which element(first or last) should be focused.
 *
 * In principle is nothing more than a wrapper around {@link focusMerge} with autofocus
 *
 * HTML markers (see {@link import('./constants').FOCUS_AUTO} constants) can control autofocus
 */
export var setFocus = function (topNode, lastNode, options) {
    if (options === void 0) { options = {}; }
    var focusable = focusMerge(topNode, lastNode);
    if (lockDisabled) {
        return;
    }
    if (focusable) {
        if (guardCount > 2) {
            // tslint:disable-next-line:no-console
            console.error('FocusLock: focus-fighting detected. Only one focus management system could be active. ' +
                'See https://github.com/theKashey/focus-lock/#focus-fighting');
            lockDisabled = true;
            setTimeout(function () {
                lockDisabled = false;
            }, 1);
            return;
        }
        guardCount++;
        focusOn(focusable.node, options.focusOptions);
        guardCount--;
    }
};
