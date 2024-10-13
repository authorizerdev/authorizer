import { isFunction } from '@chakra-ui/utils';
import * as React from 'react';

/**
 * Assigns a value to a ref function or object
 *
 * @param ref the ref to assign to
 * @param value the value
 */
function assignRef(ref, value) {
  if (ref == null) return;

  if (isFunction(ref)) {
    ref(value);
    return;
  }

  try {
    // @ts-ignore
    ref.current = value;
  } catch (error) {
    throw new Error("Cannot assign value '" + value + "' to ref '" + ref + "'");
  }
}
/**
 * Combine multiple React refs into a single ref function.
 * This is used mostly when you need to allow consumers forward refs to
 * internal components
 *
 * @param refs refs to assign to value to
 */

function mergeRefs() {
  for (var _len = arguments.length, refs = new Array(_len), _key = 0; _key < _len; _key++) {
    refs[_key] = arguments[_key];
  }

  return function (node) {
    refs.forEach(function (ref) {
      return assignRef(ref, node);
    });
  };
}

/**
 * Creates a named context, provider, and hook.
 *
 * @param options create context options
 */
function createContext(options) {
  if (options === void 0) {
    options = {};
  }

  var _options = options,
      _options$strict = _options.strict,
      strict = _options$strict === void 0 ? true : _options$strict,
      _options$errorMessage = _options.errorMessage,
      errorMessage = _options$errorMessage === void 0 ? "useContext: `context` is undefined. Seems you forgot to wrap component within the Provider" : _options$errorMessage,
      name = _options.name;
  var Context = /*#__PURE__*/React.createContext(undefined);
  Context.displayName = name;

  function useContext() {
    var context = React.useContext(Context);

    if (!context && strict) {
      var error = new Error(errorMessage);
      error.name = "ContextError";
      Error.captureStackTrace == null ? void 0 : Error.captureStackTrace(error, useContext);
      throw error;
    }

    return context;
  }

  return [Context.Provider, useContext, Context];
}

/**
 * Gets only the valid children of a component,
 * and ignores any nullish or falsy child.
 *
 * @param children the children
 */

function getValidChildren(children) {
  return React.Children.toArray(children).filter(function (child) {
    return /*#__PURE__*/React.isValidElement(child);
  });
}

export { assignRef, createContext, getValidChildren, mergeRefs };
