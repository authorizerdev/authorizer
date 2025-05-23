"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.captureFocusRestore = exports.getRelativeFocusable = exports.focusLastElement = exports.focusFirstElement = exports.focusPrevElement = exports.focusNextElement = exports.getTabbableNodes = exports.getFocusableNodes = exports.expandFocusableNodes = exports.focusSolver = exports.moveFocusInside = exports.focusIsHidden = exports.focusInside = exports.constants = void 0;
var tslib_1 = require("tslib");
var allConstants = (0, tslib_1.__importStar)(require("./constants"));
var focusInside_1 = require("./focusInside");
Object.defineProperty(exports, "focusInside", { enumerable: true, get: function () { return focusInside_1.focusInside; } });
var focusIsHidden_1 = require("./focusIsHidden");
Object.defineProperty(exports, "focusIsHidden", { enumerable: true, get: function () { return focusIsHidden_1.focusIsHidden; } });
var focusSolver_1 = require("./focusSolver");
Object.defineProperty(exports, "focusSolver", { enumerable: true, get: function () { return focusSolver_1.focusSolver; } });
var focusables_1 = require("./focusables");
Object.defineProperty(exports, "expandFocusableNodes", { enumerable: true, get: function () { return focusables_1.expandFocusableNodes; } });
var moveFocusInside_1 = require("./moveFocusInside");
Object.defineProperty(exports, "moveFocusInside", { enumerable: true, get: function () { return moveFocusInside_1.moveFocusInside; } });
var return_focus_1 = require("./return-focus");
Object.defineProperty(exports, "captureFocusRestore", { enumerable: true, get: function () { return return_focus_1.captureFocusRestore; } });
var sibling_1 = require("./sibling");
Object.defineProperty(exports, "focusNextElement", { enumerable: true, get: function () { return sibling_1.focusNextElement; } });
Object.defineProperty(exports, "focusPrevElement", { enumerable: true, get: function () { return sibling_1.focusPrevElement; } });
Object.defineProperty(exports, "getRelativeFocusable", { enumerable: true, get: function () { return sibling_1.getRelativeFocusable; } });
Object.defineProperty(exports, "focusFirstElement", { enumerable: true, get: function () { return sibling_1.focusFirstElement; } });
Object.defineProperty(exports, "focusLastElement", { enumerable: true, get: function () { return sibling_1.focusLastElement; } });
var DOMutils_1 = require("./utils/DOMutils");
Object.defineProperty(exports, "getFocusableNodes", { enumerable: true, get: function () { return DOMutils_1.getFocusableNodes; } });
Object.defineProperty(exports, "getTabbableNodes", { enumerable: true, get: function () { return DOMutils_1.getTabbableNodes; } });
/**
 * magic symbols to control focus behavior from DOM
 * see description of every particular one
 */
var constants = allConstants;
exports.constants = constants;
/**
 * @deprecated - please use {@link moveFocusInside} named export
 */
var deprecated_default_moveFocusInside = moveFocusInside_1.moveFocusInside;
exports.default = deprecated_default_moveFocusInside;
//
