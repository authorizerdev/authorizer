"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.focusLastElement = exports.focusFirstElement = exports.focusPrevElement = exports.focusNextElement = exports.getRelativeFocusable = void 0;
var commands_1 = require("./commands");
var DOMutils_1 = require("./utils/DOMutils");
var array_1 = require("./utils/array");
/**
 * for a given `element` in a given `scope` returns focusable siblings
 * @param element - base element
 * @param scope - common parent. Can be document, but better to narrow it down for performance reasons
 * @returns {prev,next} - references to a focusable element before and after
 * @returns undefined - if operation is not applicable
 */
var getRelativeFocusable = function (element, scope, useTabbables) {
    if (!element || !scope) {
        console.error('no element or scope given');
        return {};
    }
    var shards = (0, array_1.asArray)(scope);
    if (shards.every(function (shard) { return !(0, DOMutils_1.contains)(shard, element); })) {
        console.error('Active element is not contained in the scope');
        return {};
    }
    var focusables = useTabbables
        ? (0, DOMutils_1.getTabbableNodes)(shards, new Map())
        : (0, DOMutils_1.getFocusableNodes)(shards, new Map());
    var current = focusables.findIndex(function (_a) {
        var node = _a.node;
        return node === element;
    });
    if (current === -1) {
        // an edge case, when anchor element is not found
        return undefined;
    }
    return {
        prev: focusables[current - 1],
        next: focusables[current + 1],
        first: focusables[0],
        last: focusables[focusables.length - 1],
    };
};
exports.getRelativeFocusable = getRelativeFocusable;
var getBoundary = function (shards, useTabbables) {
    var set = useTabbables
        ? (0, DOMutils_1.getTabbableNodes)((0, array_1.asArray)(shards), new Map())
        : (0, DOMutils_1.getFocusableNodes)((0, array_1.asArray)(shards), new Map());
    return {
        first: set[0],
        last: set[set.length - 1],
    };
};
var defaultOptions = function (options) {
    return Object.assign({
        scope: document.body,
        cycle: true,
        onlyTabbable: true,
    }, options);
};
var moveFocus = function (fromElement, options, cb) {
    if (options === void 0) { options = {}; }
    var newOptions = defaultOptions(options);
    var solution = (0, exports.getRelativeFocusable)(fromElement, newOptions.scope, newOptions.onlyTabbable);
    if (!solution) {
        return;
    }
    var target = cb(solution, newOptions.cycle);
    if (target) {
        (0, commands_1.focusOn)(target.node, newOptions.focusOptions);
    }
};
/**
 * focuses next element in the tab-order
 * @param fromElement - common parent to scope active element search or tab cycle order
 * @param {FocusNextOptions} [options] - focus options
 */
var focusNextElement = function (fromElement, options) {
    if (options === void 0) { options = {}; }
    moveFocus(fromElement, options, function (_a, cycle) {
        var next = _a.next, first = _a.first;
        return next || (cycle && first);
    });
};
exports.focusNextElement = focusNextElement;
/**
 * focuses prev element in the tab order
 * @param fromElement - common parent to scope active element search or tab cycle order
 * @param {FocusNextOptions} [options] - focus options
 */
var focusPrevElement = function (fromElement, options) {
    if (options === void 0) { options = {}; }
    moveFocus(fromElement, options, function (_a, cycle) {
        var prev = _a.prev, last = _a.last;
        return prev || (cycle && last);
    });
};
exports.focusPrevElement = focusPrevElement;
var pickBoundary = function (scope, options, what) {
    var _a;
    var boundary = getBoundary(scope, (_a = options.onlyTabbable) !== null && _a !== void 0 ? _a : true);
    var node = boundary[what];
    if (node) {
        (0, commands_1.focusOn)(node.node, options.focusOptions);
    }
};
/**
 * focuses first element in the tab-order
 * @param {FocusNextOptions} options - focus options
 */
var focusFirstElement = function (scope, options) {
    if (options === void 0) { options = {}; }
    pickBoundary(scope, options, 'first');
};
exports.focusFirstElement = focusFirstElement;
/**
 * focuses last element in the tab order
 * @param {FocusNextOptions} options - focus options
 */
var focusLastElement = function (scope, options) {
    if (options === void 0) { options = {}; }
    pickBoundary(scope, options, 'last');
};
exports.focusLastElement = focusLastElement;
