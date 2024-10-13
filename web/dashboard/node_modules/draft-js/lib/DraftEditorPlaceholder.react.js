/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 * @format
 * 
 * @emails oncall+draft_js
 */
'use strict';

function _inheritsLoose(subClass, superClass) { subClass.prototype = Object.create(superClass.prototype); subClass.prototype.constructor = subClass; subClass.__proto__ = superClass; }

var React = require("react");

var cx = require("fbjs/lib/cx");

/**
 * This component is responsible for rendering placeholder text for the
 * `DraftEditor` component.
 *
 * Override placeholder style via CSS.
 */
var DraftEditorPlaceholder = /*#__PURE__*/function (_React$Component) {
  _inheritsLoose(DraftEditorPlaceholder, _React$Component);

  function DraftEditorPlaceholder() {
    return _React$Component.apply(this, arguments) || this;
  }

  var _proto = DraftEditorPlaceholder.prototype;

  _proto.shouldComponentUpdate = function shouldComponentUpdate(nextProps) {
    return this.props.text !== nextProps.text || this.props.editorState.getSelection().getHasFocus() !== nextProps.editorState.getSelection().getHasFocus();
  };

  _proto.render = function render() {
    var hasFocus = this.props.editorState.getSelection().getHasFocus();
    var className = cx({
      'public/DraftEditorPlaceholder/root': true,
      'public/DraftEditorPlaceholder/hasFocus': hasFocus
    });
    var contentStyle = {
      whiteSpace: 'pre-wrap'
    };
    return React.createElement("div", {
      className: className
    }, React.createElement("div", {
      className: cx('public/DraftEditorPlaceholder/inner'),
      id: this.props.accessibilityID,
      style: contentStyle
    }, this.props.text));
  };

  return DraftEditorPlaceholder;
}(React.Component);

module.exports = DraftEditorPlaceholder;