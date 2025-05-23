"use strict";

var _interopRequireDefault = require("@babel/runtime/helpers/interopRequireDefault");
var _typeof = require("@babel/runtime/helpers/typeof");
Object.defineProperty(exports, "__esModule", {
  value: true
});
Object.defineProperty(exports, "AutoFocusInside", {
  enumerable: true,
  get: function get() {
    return _AutoFocusInside["default"];
  }
});
Object.defineProperty(exports, "FocusLockUI", {
  enumerable: true,
  get: function get() {
    return _Lock["default"];
  }
});
Object.defineProperty(exports, "FreeFocusInside", {
  enumerable: true,
  get: function get() {
    return _FreeFocusInside["default"];
  }
});
Object.defineProperty(exports, "InFocusGuard", {
  enumerable: true,
  get: function get() {
    return _FocusGuard["default"];
  }
});
Object.defineProperty(exports, "MoveFocusInside", {
  enumerable: true,
  get: function get() {
    return _MoveFocusInside["default"];
  }
});
exports["default"] = void 0;
Object.defineProperty(exports, "useFocusController", {
  enumerable: true,
  get: function get() {
    return _useFocusScope.useFocusController;
  }
});
Object.defineProperty(exports, "useFocusInside", {
  enumerable: true,
  get: function get() {
    return _MoveFocusInside.useFocusInside;
  }
});
Object.defineProperty(exports, "useFocusScope", {
  enumerable: true,
  get: function get() {
    return _useFocusScope.useFocusScope;
  }
});
Object.defineProperty(exports, "useFocusState", {
  enumerable: true,
  get: function get() {
    return _useFocusState.useFocusState;
  }
});
var _Lock = _interopRequireDefault(require("./Lock"));
var _AutoFocusInside = _interopRequireDefault(require("./AutoFocusInside"));
var _MoveFocusInside = _interopRequireWildcard(require("./MoveFocusInside"));
var _FreeFocusInside = _interopRequireDefault(require("./FreeFocusInside"));
var _FocusGuard = _interopRequireDefault(require("./FocusGuard"));
var _useFocusScope = require("./use-focus-scope");
var _useFocusState = require("./use-focus-state");
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function _getRequireWildcardCache(e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != _typeof(e) && "function" != typeof e) return { "default": e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && Object.prototype.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n["default"] = e, t && t.set(e, n), n; }
var _default = exports["default"] = _Lock["default"];