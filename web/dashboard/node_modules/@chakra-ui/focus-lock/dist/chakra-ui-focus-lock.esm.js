import * as React from 'react';
import ReactFocusLock from 'react-focus-lock';
import { getAllFocusable, focus, __DEV__ } from '@chakra-ui/utils';

var FocusLock = function FocusLock(props) {
  var initialFocusRef = props.initialFocusRef,
      finalFocusRef = props.finalFocusRef,
      contentRef = props.contentRef,
      restoreFocus = props.restoreFocus,
      children = props.children,
      isDisabled = props.isDisabled,
      autoFocus = props.autoFocus,
      persistentFocus = props.persistentFocus,
      lockFocusAcrossFrames = props.lockFocusAcrossFrames;
  var onActivation = React.useCallback(function () {
    if (initialFocusRef != null && initialFocusRef.current) {
      initialFocusRef.current.focus();
    } else if (contentRef != null && contentRef.current) {
      var focusables = getAllFocusable(contentRef.current);

      if (focusables.length === 0) {
        focus(contentRef.current, {
          nextTick: true
        });
      }
    }
  }, [initialFocusRef, contentRef]);
  var onDeactivation = React.useCallback(function () {
    var _finalFocusRef$curren;

    finalFocusRef == null ? void 0 : (_finalFocusRef$curren = finalFocusRef.current) == null ? void 0 : _finalFocusRef$curren.focus();
  }, [finalFocusRef]);
  var returnFocus = restoreFocus && !finalFocusRef;
  return /*#__PURE__*/React.createElement(ReactFocusLock, {
    crossFrame: lockFocusAcrossFrames,
    persistentFocus: persistentFocus,
    autoFocus: autoFocus,
    disabled: isDisabled,
    onActivation: onActivation,
    onDeactivation: onDeactivation,
    returnFocus: returnFocus
  }, children);
};

if (__DEV__) {
  FocusLock.displayName = "FocusLock";
}

export { FocusLock, FocusLock as default };
