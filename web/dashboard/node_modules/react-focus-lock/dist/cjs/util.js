"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.deferAction = deferAction;
exports.inlineProp = exports.extractRef = void 0;
function deferAction(action) {
  setTimeout(action, 1);
}
var inlineProp = exports.inlineProp = function inlineProp(name, value) {
  var obj = {};
  obj[name] = value;
  return obj;
};
var extractRef = exports.extractRef = function extractRef(ref) {
  return ref && 'current' in ref ? ref.current : ref;
};