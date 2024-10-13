import { forwardRef, useMultiStyleConfig, omitThemingProps, StylesProvider, chakra, useStyles } from '@chakra-ui/system';
import { focus, normalizeEventKey, isEmpty, getRelatedTarget, contains, ariaAttr, callAllHandlers, cx, runIfFn, __DEV__ } from '@chakra-ui/utils';
import { mergeRefs, createContext } from '@chakra-ui/react-utils';
import * as React from 'react';
import { useState, useRef, useCallback } from 'react';
import { useControllableState, useFocusOnPointerDown, useUpdateEffect } from '@chakra-ui/hooks';

function _extends() {
  _extends = Object.assign || function (target) {
    for (var i = 1; i < arguments.length; i++) {
      var source = arguments[i];

      for (var key in source) {
        if (Object.prototype.hasOwnProperty.call(source, key)) {
          target[key] = source[key];
        }
      }
    }

    return target;
  };

  return _extends.apply(this, arguments);
}

function _objectWithoutPropertiesLoose(source, excluded) {
  if (source == null) return {};
  var target = {};
  var sourceKeys = Object.keys(source);
  var key, i;

  for (i = 0; i < sourceKeys.length; i++) {
    key = sourceKeys[i];
    if (excluded.indexOf(key) >= 0) continue;
    target[key] = source[key];
  }

  return target;
}

var _excluded$1 = ["onChange", "onCancel", "onSubmit", "value", "isDisabled", "defaultValue", "startWithEditView", "isPreviewFocusable", "submitOnBlur", "selectAllOnFocus", "placeholder", "onEdit"];

/**
 * React hook for managing the inline renaming of some text.
 *
 * @see Docs https://chakra-ui.com/editable
 */
function useEditable(props) {
  if (props === void 0) {
    props = {};
  }

  var _props = props,
      onChangeProp = _props.onChange,
      onCancelProp = _props.onCancel,
      onSubmitProp = _props.onSubmit,
      valueProp = _props.value,
      isDisabled = _props.isDisabled,
      defaultValue = _props.defaultValue,
      startWithEditView = _props.startWithEditView,
      _props$isPreviewFocus = _props.isPreviewFocusable,
      isPreviewFocusable = _props$isPreviewFocus === void 0 ? true : _props$isPreviewFocus,
      _props$submitOnBlur = _props.submitOnBlur,
      submitOnBlur = _props$submitOnBlur === void 0 ? true : _props$submitOnBlur,
      _props$selectAllOnFoc = _props.selectAllOnFocus,
      selectAllOnFocus = _props$selectAllOnFoc === void 0 ? true : _props$selectAllOnFoc,
      placeholder = _props.placeholder,
      onEditProp = _props.onEdit,
      htmlProps = _objectWithoutPropertiesLoose(_props, _excluded$1);

  var defaultIsEditing = Boolean(startWithEditView && !isDisabled);

  var _useState = useState(defaultIsEditing),
      isEditing = _useState[0],
      setIsEditing = _useState[1];

  var _useControllableState = useControllableState({
    defaultValue: defaultValue || "",
    value: valueProp,
    onChange: onChangeProp
  }),
      value = _useControllableState[0],
      setValue = _useControllableState[1];
  /**
   * Keep track of the previous value, so if users
   * presses `cancel`, we can revert to it.
   */


  var _useState2 = useState(value),
      prevValue = _useState2[0],
      setPrevValue = _useState2[1];
  /**
   * Ref to help focus the input in edit mode
   */


  var inputRef = useRef(null);
  var previewRef = useRef(null);
  var editButtonRef = useRef(null);
  var cancelButtonRef = useRef(null);
  var submitButtonRef = useRef(null);
  useFocusOnPointerDown({
    ref: inputRef,
    enabled: isEditing,
    elements: [cancelButtonRef, submitButtonRef]
  });
  var isInteractive = !isEditing || !isDisabled;
  useUpdateEffect(function () {
    if (!isEditing) {
      focus(editButtonRef.current);
      return;
    }

    focus(inputRef.current, {
      selectTextIfInput: selectAllOnFocus
    });
    onEditProp == null ? void 0 : onEditProp();
  }, [isEditing, onEditProp, selectAllOnFocus]);
  var onEdit = useCallback(function () {
    if (isInteractive) {
      setIsEditing(true);
    }
  }, [isInteractive]);
  var onCancel = useCallback(function () {
    setIsEditing(false);
    setValue(prevValue);
    onCancelProp == null ? void 0 : onCancelProp(prevValue);
  }, [onCancelProp, setValue, prevValue]);
  var onSubmit = useCallback(function () {
    setIsEditing(false);
    setPrevValue(value);
    onSubmitProp == null ? void 0 : onSubmitProp(value);
  }, [value, onSubmitProp]);
  var onChange = useCallback(function (event) {
    setValue(event.target.value);
  }, [setValue]);
  var onKeyDown = useCallback(function (event) {
    var eventKey = normalizeEventKey(event);
    var keyMap = {
      Escape: onCancel,
      Enter: function Enter(event) {
        if (!event.shiftKey && !event.metaKey) {
          onSubmit();
        }
      }
    };
    var action = keyMap[eventKey];

    if (action) {
      event.preventDefault();
      action(event);
    }
  }, [onCancel, onSubmit]);
  var isValueEmpty = isEmpty(value);
  var onBlur = useCallback(function (event) {
    var relatedTarget = getRelatedTarget(event);
    var targetIsCancel = contains(cancelButtonRef.current, relatedTarget);
    var targetIsSubmit = contains(submitButtonRef.current, relatedTarget);
    var isValidBlur = !targetIsCancel && !targetIsSubmit;

    if (isValidBlur && submitOnBlur) {
      onSubmit();
    }
  }, [submitOnBlur, onSubmit]);
  var getPreviewProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    var tabIndex = isInteractive && isPreviewFocusable ? 0 : undefined;
    return _extends({}, props, {
      ref: mergeRefs(ref, previewRef),
      children: isValueEmpty ? placeholder : value,
      hidden: isEditing,
      "aria-disabled": ariaAttr(isDisabled),
      tabIndex: tabIndex,
      onFocus: callAllHandlers(props.onFocus, onEdit)
    });
  }, [isDisabled, isEditing, isInteractive, isPreviewFocusable, isValueEmpty, onEdit, placeholder, value]);
  var getInputProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      hidden: !isEditing,
      placeholder: placeholder,
      ref: mergeRefs(ref, inputRef),
      disabled: isDisabled,
      "aria-disabled": ariaAttr(isDisabled),
      value: value,
      onBlur: callAllHandlers(props.onBlur, onBlur),
      onChange: callAllHandlers(props.onChange, onChange),
      onKeyDown: callAllHandlers(props.onKeyDown, onKeyDown)
    });
  }, [isDisabled, isEditing, onBlur, onChange, onKeyDown, placeholder, value]);
  var getEditButtonProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({
      "aria-label": "Edit"
    }, props, {
      type: "button",
      onClick: callAllHandlers(props.onClick, onEdit),
      ref: mergeRefs(ref, editButtonRef)
    });
  }, [onEdit]);
  var getSubmitButtonProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({}, props, {
      "aria-label": "Submit",
      ref: mergeRefs(submitButtonRef, ref),
      type: "button",
      onClick: callAllHandlers(props.onClick, onSubmit)
    });
  }, [onSubmit]);
  var getCancelButtonProps = useCallback(function (props, ref) {
    if (props === void 0) {
      props = {};
    }

    if (ref === void 0) {
      ref = null;
    }

    return _extends({
      "aria-label": "Cancel",
      id: "cancel"
    }, props, {
      ref: mergeRefs(cancelButtonRef, ref),
      type: "button",
      onClick: callAllHandlers(props.onClick, onCancel)
    });
  }, [onCancel]);
  return {
    isEditing: isEditing,
    isDisabled: isDisabled,
    isValueEmpty: isValueEmpty,
    value: value,
    onEdit: onEdit,
    onCancel: onCancel,
    onSubmit: onSubmit,
    getPreviewProps: getPreviewProps,
    getInputProps: getInputProps,
    getEditButtonProps: getEditButtonProps,
    getSubmitButtonProps: getSubmitButtonProps,
    getCancelButtonProps: getCancelButtonProps,
    htmlProps: htmlProps
  };
}

var _excluded = ["htmlProps"];

var _createContext = createContext({
  name: "EditableContext",
  errorMessage: "useEditableContext: context is undefined. Seems you forgot to wrap the editable components in `<Editable />`"
}),
    EditableProvider = _createContext[0],
    useEditableContext = _createContext[1];

/**
 * Editable
 *
 * The wrapper that provides context and logic for all editable
 * components. It renders a `div`
 */
var Editable = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useMultiStyleConfig("Editable", props);
  var ownProps = omitThemingProps(props);

  var _useEditable = useEditable(ownProps),
      htmlProps = _useEditable.htmlProps,
      context = _objectWithoutPropertiesLoose(_useEditable, _excluded);

  var isEditing = context.isEditing,
      onSubmit = context.onSubmit,
      onCancel = context.onCancel,
      onEdit = context.onEdit;

  var _className = cx("chakra-editable", props.className);

  var children = runIfFn(props.children, {
    isEditing: isEditing,
    onSubmit: onSubmit,
    onCancel: onCancel,
    onEdit: onEdit
  });
  return /*#__PURE__*/React.createElement(EditableProvider, {
    value: context
  }, /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref
  }, htmlProps, {
    className: _className
  }), children)));
});

if (__DEV__) {
  Editable.displayName = "Editable";
}

var commonStyles = {
  fontSize: "inherit",
  fontWeight: "inherit",
  textAlign: "inherit",
  bg: "transparent"
};

/**
 * EditablePreview
 *
 * The `span` used to display the final value, in the `preview` mode
 */
var EditablePreview = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useEditableContext = useEditableContext(),
      getPreviewProps = _useEditableContext.getPreviewProps;

  var styles = useStyles();
  var previewProps = getPreviewProps(props, ref);

  var _className = cx("chakra-editable__preview", props.className);

  return /*#__PURE__*/React.createElement(chakra.span, _extends({}, previewProps, {
    __css: _extends({
      cursor: "text",
      display: "inline-block"
    }, commonStyles, styles.preview),
    className: _className
  }));
});

if (__DEV__) {
  EditablePreview.displayName = "EditablePreview";
}

/**
 * EditableInput
 *
 * The input used in the `edit` mode
 */
var EditableInput = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useEditableContext2 = useEditableContext(),
      getInputProps = _useEditableContext2.getInputProps;

  var styles = useStyles();
  var inputProps = getInputProps(props, ref);

  var _className = cx("chakra-editable__input", props.className);

  return /*#__PURE__*/React.createElement(chakra.input, _extends({}, inputProps, {
    __css: _extends({
      outline: 0
    }, commonStyles, styles.input),
    className: _className
  }));
});

if (__DEV__) {
  EditableInput.displayName = "EditableInput";
}
/**
 * React hook use to gain access to the editable state and actions.
 */


function useEditableState() {
  var _useEditableContext3 = useEditableContext(),
      isEditing = _useEditableContext3.isEditing,
      onSubmit = _useEditableContext3.onSubmit,
      onCancel = _useEditableContext3.onCancel,
      onEdit = _useEditableContext3.onEdit,
      isDisabled = _useEditableContext3.isDisabled;

  return {
    isEditing: isEditing,
    onSubmit: onSubmit,
    onCancel: onCancel,
    onEdit: onEdit,
    isDisabled: isDisabled
  };
}
/**
 * React hook use to create controls for the editable component
 */

function useEditableControls() {
  var _useEditableContext4 = useEditableContext(),
      isEditing = _useEditableContext4.isEditing,
      getEditButtonProps = _useEditableContext4.getEditButtonProps,
      getCancelButtonProps = _useEditableContext4.getCancelButtonProps,
      getSubmitButtonProps = _useEditableContext4.getSubmitButtonProps;

  return {
    isEditing: isEditing,
    getEditButtonProps: getEditButtonProps,
    getCancelButtonProps: getCancelButtonProps,
    getSubmitButtonProps: getSubmitButtonProps
  };
}

export { Editable, EditableInput, EditablePreview, useEditable, useEditableControls, useEditableState };
