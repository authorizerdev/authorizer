'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var provider = require('@chakra-ui/provider');
var theme = require('@chakra-ui/theme');
var utils = require('@chakra-ui/utils');
var accordion = require('@chakra-ui/accordion');
var alert = require('@chakra-ui/alert');
var avatar = require('@chakra-ui/avatar');
var breadcrumb = require('@chakra-ui/breadcrumb');
var button = require('@chakra-ui/button');
var checkbox = require('@chakra-ui/checkbox');
var closeButton = require('@chakra-ui/close-button');
var counter = require('@chakra-ui/counter');
var cssReset = require('@chakra-ui/css-reset');
var editable = require('@chakra-ui/editable');
var formControl = require('@chakra-ui/form-control');
var controlBox = require('@chakra-ui/control-box');
var hooks = require('@chakra-ui/hooks');
var icon = require('@chakra-ui/icon');
var image = require('@chakra-ui/image');
var input = require('@chakra-ui/input');
var layout = require('@chakra-ui/layout');
var mediaQuery = require('@chakra-ui/media-query');
var table = require('@chakra-ui/table');
var menu = require('@chakra-ui/menu');
var modal = require('@chakra-ui/modal');
var numberInput = require('@chakra-ui/number-input');
var pinInput = require('@chakra-ui/pin-input');
var popover = require('@chakra-ui/popover');
var popper = require('@chakra-ui/popper');
var portal = require('@chakra-ui/portal');
var progress = require('@chakra-ui/progress');
var radio = require('@chakra-ui/radio');
var select = require('@chakra-ui/select');
var skeleton = require('@chakra-ui/skeleton');
var slider = require('@chakra-ui/slider');
var spinner = require('@chakra-ui/spinner');
var stat = require('@chakra-ui/stat');
var _switch = require('@chakra-ui/switch');
var system = require('@chakra-ui/system');
var tabs = require('@chakra-ui/tabs');
var tag = require('@chakra-ui/tag');
var textarea = require('@chakra-ui/textarea');
var toast = require('@chakra-ui/toast');
var tooltip = require('@chakra-ui/tooltip');
var transition = require('@chakra-ui/transition');
var visuallyHidden = require('@chakra-ui/visually-hidden');

var ChakraProvider = provider.ChakraProvider;
ChakraProvider.defaultProps = {
  theme: theme.theme
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

  if (theme.isChakraTheme(baseTheme) && // this ensures backward compatibility
  // previously only `extendTheme(override, baseTheme?)` was allowed
  overrides.length > 1) {
    overrides = overrides.slice(0, overrides.length - 1);
  } else {
    baseTheme = theme.theme;
  }

  return utils.pipe.apply(void 0, overrides.map(function (extension) {
    return function (prevTheme) {
      return utils.isFunction(extension) ? extension(prevTheme) : mergeThemeOverride(prevTheme, extension);
    };
  }))(baseTheme);
}
function mergeThemeOverride() {
  for (var _len2 = arguments.length, overrides = new Array(_len2), _key2 = 0; _key2 < _len2; _key2++) {
    overrides[_key2] = arguments[_key2];
  }

  return utils.mergeWith.apply(void 0, [{}].concat(overrides, [mergeThemeCustomizer]));
}

function mergeThemeCustomizer(source, override, key, object) {
  if ((utils.isFunction(source) || utils.isFunction(override)) && Object.prototype.hasOwnProperty.call(object, key)) {
    return function () {
      var sourceValue = utils.isFunction(source) ? source.apply(void 0, arguments) : source;
      var overrideValue = utils.isFunction(override) ? override.apply(void 0, arguments) : override;
      return utils.mergeWith({}, sourceValue, overrideValue, mergeThemeCustomizer);
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
    } else if (utils.isObject(components)) {
      names = Object.keys(components);
    }

    return mergeThemeOverride(theme, {
      components: utils.fromEntries(names.map(function (componentName) {
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
    } else if (utils.isObject(components)) {
      names = Object.keys(components);
    }

    return mergeThemeOverride(theme, {
      components: utils.fromEntries(names.map(function (componentName) {
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
    } else if (utils.isObject(components)) {
      names = Object.keys(components);
    }

    return mergeThemeOverride(theme, {
      components: utils.fromEntries(names.map(function (componentName) {
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
    return mergeThemeOverride(utils.pipe.apply(void 0, fns)(theme));
  };
}

exports.ChakraProvider = ChakraProvider;
exports.extendTheme = extendTheme;
exports.mergeThemeOverride = mergeThemeOverride;
exports.withDefaultColorScheme = withDefaultColorScheme;
exports.withDefaultProps = withDefaultProps;
exports.withDefaultSize = withDefaultSize;
exports.withDefaultVariant = withDefaultVariant;
Object.keys(theme).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return theme[k]; }
  });
});
Object.keys(accordion).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return accordion[k]; }
  });
});
Object.keys(alert).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return alert[k]; }
  });
});
Object.keys(avatar).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return avatar[k]; }
  });
});
Object.keys(breadcrumb).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return breadcrumb[k]; }
  });
});
Object.keys(button).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return button[k]; }
  });
});
Object.keys(checkbox).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return checkbox[k]; }
  });
});
Object.keys(closeButton).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return closeButton[k]; }
  });
});
Object.keys(counter).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return counter[k]; }
  });
});
Object.keys(cssReset).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return cssReset[k]; }
  });
});
Object.keys(editable).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return editable[k]; }
  });
});
Object.keys(formControl).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return formControl[k]; }
  });
});
Object.keys(controlBox).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return controlBox[k]; }
  });
});
Object.keys(hooks).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return hooks[k]; }
  });
});
Object.keys(icon).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return icon[k]; }
  });
});
Object.keys(image).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return image[k]; }
  });
});
Object.keys(input).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return input[k]; }
  });
});
Object.keys(layout).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return layout[k]; }
  });
});
Object.keys(mediaQuery).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return mediaQuery[k]; }
  });
});
Object.keys(table).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return table[k]; }
  });
});
Object.keys(menu).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return menu[k]; }
  });
});
Object.keys(modal).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return modal[k]; }
  });
});
Object.keys(numberInput).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return numberInput[k]; }
  });
});
Object.keys(pinInput).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return pinInput[k]; }
  });
});
Object.keys(popover).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return popover[k]; }
  });
});
Object.keys(popper).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return popper[k]; }
  });
});
Object.keys(portal).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return portal[k]; }
  });
});
Object.keys(progress).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return progress[k]; }
  });
});
Object.keys(radio).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return radio[k]; }
  });
});
Object.keys(select).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return select[k]; }
  });
});
Object.keys(skeleton).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return skeleton[k]; }
  });
});
Object.keys(slider).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return slider[k]; }
  });
});
Object.keys(spinner).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return spinner[k]; }
  });
});
Object.keys(stat).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return stat[k]; }
  });
});
Object.keys(_switch).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return _switch[k]; }
  });
});
Object.keys(system).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return system[k]; }
  });
});
Object.keys(tabs).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return tabs[k]; }
  });
});
Object.keys(tag).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return tag[k]; }
  });
});
Object.keys(textarea).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return textarea[k]; }
  });
});
Object.keys(toast).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return toast[k]; }
  });
});
Object.keys(tooltip).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return tooltip[k]; }
  });
});
Object.keys(transition).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return transition[k]; }
  });
});
Object.keys(visuallyHidden).forEach(function (k) {
  if (k !== 'default' && !exports.hasOwnProperty(k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: function () { return visuallyHidden[k]; }
  });
});
