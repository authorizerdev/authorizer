import { forwardRef, chakra } from '@chakra-ui/system';
import { omit, __DEV__ } from '@chakra-ui/utils';
import * as React from 'react';
import { useState, useEffect, useRef, useCallback } from 'react';
import { useSafeLayoutEffect } from '@chakra-ui/hooks';

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

/**
 * React hook that loads an image in the browser,
 * and let's us know the `status` so we can show image
 * fallback if it is still `pending`
 *
 * @returns the status of the image loading progress
 *
 * @example
 *
 * ```jsx
 * function App(){
 *   const status = useImage({ src: "image.png" })
 *   return status === "loaded" ? <img src="image.png" /> : <Placeholder />
 * }
 * ```
 */
function useImage(props) {
  var loading = props.loading,
      src = props.src,
      srcSet = props.srcSet,
      onLoad = props.onLoad,
      onError = props.onError,
      crossOrigin = props.crossOrigin,
      sizes = props.sizes,
      ignoreFallback = props.ignoreFallback;

  var _useState = useState("pending"),
      status = _useState[0],
      setStatus = _useState[1];

  useEffect(function () {
    setStatus(src ? "loading" : "pending");
  }, [src]);
  var imageRef = useRef();
  var load = useCallback(function () {
    if (!src) return;
    flush();
    var img = new Image();
    img.src = src;
    if (crossOrigin) img.crossOrigin = crossOrigin;
    if (srcSet) img.srcset = srcSet;
    if (sizes) img.sizes = sizes;
    if (loading) img.loading = loading;

    img.onload = function (event) {
      flush();
      setStatus("loaded");
      onLoad == null ? void 0 : onLoad(event);
    };

    img.onerror = function (error) {
      flush();
      setStatus("failed");
      onError == null ? void 0 : onError(error);
    };

    imageRef.current = img;
  }, [src, crossOrigin, srcSet, sizes, onLoad, onError, loading]);

  var flush = function flush() {
    if (imageRef.current) {
      imageRef.current.onload = null;
      imageRef.current.onerror = null;
      imageRef.current = null;
    }
  };

  useSafeLayoutEffect(function () {
    /**
     * If user opts out of the fallback/placeholder
     * logic, let's bail out.
     */
    if (ignoreFallback) return undefined;

    if (status === "loading") {
      load();
    }

    return function () {
      flush();
    };
  }, [status, load, ignoreFallback]);
  /**
   * If user opts out of the fallback/placeholder
   * logic, let's just return 'loaded'
   */

  return ignoreFallback ? "loaded" : status;
}

var _excluded = ["htmlWidth", "htmlHeight", "alt"],
    _excluded2 = ["fallbackSrc", "fallback", "src", "srcSet", "align", "fit", "loading", "ignoreFallback", "crossOrigin"];
var NativeImage = /*#__PURE__*/React.forwardRef(function (props, ref) {
  var htmlWidth = props.htmlWidth,
      htmlHeight = props.htmlHeight,
      alt = props.alt,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  return /*#__PURE__*/React.createElement("img", _extends({
    width: htmlWidth,
    height: htmlHeight,
    ref: ref,
    alt: alt
  }, rest));
});

/**
 * React component that renders an image with support
 * for fallbacks
 *
 * @see Docs https://chakra-ui.com/image
 */
var Image$1 = /*#__PURE__*/forwardRef(function (props, ref) {
  var fallbackSrc = props.fallbackSrc,
      fallback = props.fallback,
      src = props.src,
      srcSet = props.srcSet,
      align = props.align,
      fit = props.fit,
      loading = props.loading,
      ignoreFallback = props.ignoreFallback,
      crossOrigin = props.crossOrigin,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);
  /**
   * Defer to native `img` tag if `loading` prop is passed
   * @see https://github.com/chakra-ui/chakra-ui/issues/1027
   */


  var shouldIgnore = loading != null || ignoreFallback || fallbackSrc === undefined && fallback === undefined; // if the user doesn't provide any kind of fallback we should ignore it

  var status = useImage(_extends({}, props, {
    ignoreFallback: shouldIgnore
  }));

  var shared = _extends({
    ref: ref,
    objectFit: fit,
    objectPosition: align
  }, shouldIgnore ? rest : omit(rest, ["onError", "onLoad"]));

  if (status !== "loaded") {
    /**
     * If user passed a custom fallback component,
     * let's render it here.
     */
    if (fallback) return fallback;
    return /*#__PURE__*/React.createElement(chakra.img, _extends({
      as: NativeImage,
      className: "chakra-image__placeholder",
      src: fallbackSrc
    }, shared));
  }

  return /*#__PURE__*/React.createElement(chakra.img, _extends({
    as: NativeImage,
    src: src,
    srcSet: srcSet,
    crossOrigin: crossOrigin,
    loading: loading,
    className: "chakra-image"
  }, shared));
});

/**
 * Fallback component for most SSR users who want to use the native `img` with
 * support for chakra props
 */
var Img = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(chakra.img, _extends({
    ref: ref,
    as: NativeImage,
    className: "chakra-image"
  }, props));
});

if (__DEV__) {
  Image$1.displayName = "Image";
}

export { Image$1 as Image, Img, useImage };
