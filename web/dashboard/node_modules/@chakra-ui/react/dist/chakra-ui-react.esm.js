import { ChakraProvider as ChakraProvider$1 } from '@chakra-ui/provider';
import { theme as theme$1, isChakraTheme } from '@chakra-ui/theme';
export * from '@chakra-ui/theme';
import { pipe, isFunction, mergeWith, isObject, fromEntries } from '@chakra-ui/utils';
export * from '@chakra-ui/accordion';
export * from '@chakra-ui/alert';
export * from '@chakra-ui/avatar';
export * from '@chakra-ui/breadcrumb';
export * from '@chakra-ui/button';
export * from '@chakra-ui/checkbox';
export * from '@chakra-ui/close-button';
export * from '@chakra-ui/counter';
export * from '@chakra-ui/css-reset';
export * from '@chakra-ui/editable';
export * from '@chakra-ui/form-control';
export * from '@chakra-ui/control-box';
export * from '@chakra-ui/hooks';
export * from '@chakra-ui/icon';
export * from '@chakra-ui/image';
export * from '@chakra-ui/input';
export * from '@chakra-ui/layout';
export * from '@chakra-ui/media-query';
export * from '@chakra-ui/table';
export * from '@chakra-ui/menu';
export * from '@chakra-ui/modal';
export * from '@chakra-ui/number-input';
export * from '@chakra-ui/pin-input';
export * from '@chakra-ui/popover';
export * from '@chakra-ui/popper';
export * from '@chakra-ui/portal';
export * from '@chakra-ui/progress';
export * from '@chakra-ui/radio';
export * from '@chakra-ui/select';
export * from '@chakra-ui/skeleton';
export * from '@chakra-ui/slider';
export * from '@chakra-ui/spinner';
export * from '@chakra-ui/stat';
export * from '@chakra-ui/switch';
export * from '@chakra-ui/system';
export * from '@chakra-ui/tabs';
export * from '@chakra-ui/tag';
export * from '@chakra-ui/textarea';
export * from '@chakra-ui/toast';
export * from '@chakra-ui/tooltip';
export * from '@chakra-ui/transition';
export * from '@chakra-ui/visually-hidden';

var ChakraProvider = ChakraProvider$1;
ChakraProvider.defaultProps = {
  theme: theme$1
};

/**
 * NOTE: This got too complex to manage and it's not worth the extra complexity.
 * We'll re-evaluate this API in the future releases.
 *
 * Function to override or customize the Chakra UI theme conveniently.
 * First extension overrides the baseTheme and following extensions override the preceding extensions.
 *
 * @example:
 * import { theme as baseTheme, extendTheme, withDefaultColorScheme } from '@chakra-ui/react'
 *
 * const customTheme = extendTheme(
 *   {
 *     colors: {
 *       brand: {
 *         500: "#b4d455",
 *       },
 *     },
 *   },
 *   withDefaultColorScheme({ colorScheme: "red" }),
 *   baseTheme // optional
 * )
 */
function extendTheme() {
  for (var _len = arguments.length, extensions = new Array(_len), _key = 0; _key < _len; _key++) {
    extensions[_key] = arguments[_key];
  }

  var overrides = [].concat(extensions);
  var baseTheme = extensions[extensions.length - 1];

  if (isChakraTheme(baseTheme) && // this ensures backward compatibility
  // previously only `extendTheme(override, baseTheme?)` was allowed
  overrides.length > 1) {
    overrides = overrides.slice(0, overrides.length - 1);
  } else {
    baseTheme = theme$1;
  }

  return pipe.apply(void 0, overrides.map(function (extension) {
    return function (prevTheme) {
      return isFunction(extension) ? extension(prevTheme) : mergeThemeOverride(prevTheme, extension);
    };
  }))(baseTheme);
}
function mergeThemeOverride() {
  for (var _len2 = arguments.length, overrides = new Array(_len2), _key2 = 0; _key2 < _len2; _key2++) {
    overrides[_key2] = arguments[_key2];
  }

  return mergeWith.apply(void 0, [{}].concat(overrides, [mergeThemeCustomizer]));
}

function mergeThemeCustomizer(source, override, key, object) {
  if ((isFunction(source) || isFunction(override)) && Object.prototype.hasOwnProperty.call(object, key)) {
    return function () {
      var sourceValue = isFunction(source) ? source.apply(void 0, arguments) : source;
      var overrideValue = isFunction(override) ? override.apply(void 0, arguments) : override;
      return mergeWith({}, sourceValue, overrideValue, mergeThemeCustomizer);
    };
  } // fallback to default behaviour


  return undefined;
}

function withDefaultColorScheme(_ref) {
  var colorScheme = _ref.colorScheme,
      components = _ref.components;
  return function (theme) {
    var names = Object.keys(theme.components || {});

    if (Array.isArray(components)) {
      names = components;
    } else if (isObject(components)) {
      names = Object.keys(components);
    }

    return mergeThemeOverride(theme, {
      components: fromEntries(names.map(function (componentName) {
        var withColorScheme = {
          defaultProps: {
            colorScheme: colorScheme
          }
        };
        return [componentName, withColorScheme];
      }))
    });
  };
}

function withDefaultSize(_ref) {
  var size = _ref.size,
      components = _ref.components;
  return function (theme) {
    var names = Object.keys(theme.components || {});

    if (Array.isArray(components)) {
      names = components;
    } else if (isObject(components)) {
      names = Object.keys(components);
    }

    return mergeThemeOverride(theme, {
      components: fromEntries(names.map(function (componentName) {
        var withSize = {
          defaultProps: {
            size: size
          }
        };
        return [componentName, withSize];
      }))
    });
  };
}

function withDefaultVariant(_ref) {
  var variant = _ref.variant,
      components = _ref.components;
  return function (theme) {
    var names = Object.keys(theme.components || {});

    if (Array.isArray(components)) {
      names = components;
    } else if (isObject(components)) {
      names = Object.keys(components);
    }

    return mergeThemeOverride(theme, {
      components: fromEntries(names.map(function (componentName) {
        var withVariant = {
          defaultProps: {
            variant: variant
          }
        };
        return [componentName, withVariant];
      }))
    });
  };
}

function withDefaultProps(_ref) {
  var _ref$defaultProps = _ref.defaultProps,
      colorScheme = _ref$defaultProps.colorScheme,
      variant = _ref$defaultProps.variant,
      size = _ref$defaultProps.size,
      components = _ref.components;

  var identity = function identity(t) {
    return t;
  };

  var fns = [colorScheme ? withDefaultColorScheme({
    colorScheme: colorScheme,
    components: components
  }) : identity, size ? withDefaultSize({
    size: size,
    components: components
  }) : identity, variant ? withDefaultVariant({
    variant: variant,
    components: components
  }) : identity];
  return function (theme) {
    return mergeThemeOverride(pipe.apply(void 0, fns)(theme));
  };
}

export { ChakraProvider, extendTheme, mergeThemeOverride, withDefaultColorScheme, withDefaultProps, withDefaultSize, withDefaultVariant };
