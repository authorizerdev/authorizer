'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');

/**
 * Styles to visually hide an element
 * but make it accessible to screen-readers
 */
var visuallyHiddenStyle = {
  border: "0px",
  clip: "rect(0px, 0px, 0px, 0px)",
  height: "1px",
  width: "1px",
  margin: "-1px",
  padding: "0px",
  overflow: "hidden",
  whiteSpace: "nowrap",
  position: "absolute"
};
/**
 * Visually hidden component used to hide
 * elements on screen
 */

var VisuallyHidden = system.chakra("span", {
  baseStyle: visuallyHiddenStyle
});

if (utils.__DEV__) {
  VisuallyHidden.displayName = "VisuallyHidden";
}
/**
 * Visually hidden input component for designing
 * custom input components using the html `input`
 * as a proxy
 */


var VisuallyHiddenInput = system.chakra("input", {
  baseStyle: visuallyHiddenStyle
});

if (utils.__DEV__) {
  VisuallyHiddenInput.displayName = "VisuallyHiddenInput";
}

var VisuallyHidden$1 = VisuallyHidden;

exports.VisuallyHidden = VisuallyHidden;
exports.VisuallyHiddenInput = VisuallyHiddenInput;
exports["default"] = VisuallyHidden$1;
exports.visuallyHiddenStyle = visuallyHiddenStyle;
