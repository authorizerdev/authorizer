import { focusOn } from './commands';
import { getTabbableNodes, contains, getFocusableNodes } from './utils/DOMutils';
import { asArray } from './utils/array';
/**
 * for a given `element` in a given `scope` returns focusable siblings
 * @param element - base element
 * @param scope - common parent. Can be document, but better to narrow it down for performance reasons
 * @returns {prev,next} - references to a focusable element before and after
 * @returns undefined - if operation is not applicable
 */
export var getRelativeFocusable = function (element, scope, useTabbables) {
    if (!element || !scope) {
        console.error('no element or scope given');
        return {};
    }
    var shards = asArray(scope);
    if (shards.every(function (shard) { return !contains(shard, element); })) {
        console.error('Active element is not contained in the scope');
        return {};
    }
    var focusables = useTabbables
        ? getTabbableNodes(shards, new Map())
        : getFocusableNodes(shards, new Map());
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
var getBoundary = function (shards, useTabbables) {
    var set = useTabbables
        ? getTabbableNodes(asArray(shards), new Map())
        : getFocusableNodes(asArray(shards), new Map());
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
    var solution = getRelativeFocusable(fromElement, newOptions.scope, newOptions.onlyTabbable);
    if (!solution) {
        return;
    }
    var target = cb(solution, newOptions.cycle);
    if (target) {
        focusOn(target.node, newOptions.focusOptions);
    }
};
/**
 * focuses next element in the tab-order
 * @param fromElement - common parent to scope active element search or tab cycle order
 * @param {FocusNextOptions} [options] - focus options
 */
export var focusNextElement = function (fromElement, options) {
    if (options === void 0) { options = {}; }
    moveFocus(fromElement, options, function (_a, cycle) {
        var next = _a.next, first = _a.first;
        return next || (cycle && first);
    });
};
/**
 * focuses prev element in the tab order
 * @param fromElement - common parent to scope active element search or tab cycle order
 * @param {FocusNextOptions} [options] - focus options
 */
export var focusPrevElement = function (fromElement, options) {
    if (options === void 0) { options = {}; }
    moveFocus(fromElement, options, function (_a, cycle) {
        var prev = _a.prev, last = _a.last;
        return prev || (cycle && last);
    });
};
var pickBoundary = function (scope, options, what) {
    var _a;
    var boundary = getBoundary(scope, (_a = options.onlyTabbable) !== null && _a !== void 0 ? _a : true);
    var node = boundary[what];
    if (node) {
        focusOn(node.node, options.focusOptions);
    }
};
/**
 * focuses first element in the tab-order
 * @param {FocusNextOptions} options - focus options
 */
export var focusFirstElement = function (scope, options) {
    if (options === void 0) { options = {}; }
    pickBoundary(scope, options, 'first');
};
/**
 * focuses last element in the tab order
 * @param {FocusNextOptions} options - focus options
 */
export var focusLastElement = function (scope, options) {
    if (options === void 0) { options = {}; }
    pickBoundary(scope, options, 'last');
};
