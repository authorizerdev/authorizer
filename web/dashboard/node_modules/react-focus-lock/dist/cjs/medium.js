"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.mediumSidecar = exports.mediumFocus = exports.mediumEffect = exports.mediumBlur = void 0;
var _useSidecar = require("use-sidecar");
var mediumFocus = exports.mediumFocus = (0, _useSidecar.createMedium)({}, function (_ref) {
  var target = _ref.target,
    currentTarget = _ref.currentTarget;
  return {
    target: target,
    currentTarget: currentTarget
  };
});
var mediumBlur = exports.mediumBlur = (0, _useSidecar.createMedium)();
var mediumEffect = exports.mediumEffect = (0, _useSidecar.createMedium)();
var mediumSidecar = exports.mediumSidecar = (0, _useSidecar.createSidecarMedium)({
  async: true,
  ssr: typeof document !== 'undefined'
});