import { TinyColor, readability, isReadable, random } from '@ctrl/tinycolor';
import { memoizedGet, isEmptyObject, warn, fromEntries, isObject } from '@chakra-ui/utils';
export { runIfFn } from '@chakra-ui/utils';

/**
 * Get the color raw value from theme
 * @param theme - the theme object
 * @param color - the color path ("green.200")
 * @param fallback - the fallback color
 */

var getColor = function getColor(theme, color, fallback) {
  var hex = memoizedGet(theme, "colors." + color, color);

  var _TinyColor = new TinyColor(hex),
      isValid = _TinyColor.isValid;

  return isValid ? hex : fallback;
};
/**
 * Determines if the tone of given color is "light" or "dark"
 * @param color - the color in hex, rgb, or hsl
 */

var tone = function tone(color) {
  return function (theme) {
    var hex = getColor(theme, color);
    var isDark = new TinyColor(hex).isDark();
    return isDark ? "dark" : "light";
  };
};
/**
 * Determines if a color tone is "dark"
 * @param color - the color in hex, rgb, or hsl
 */

var isDark = function isDark(color) {
  return function (theme) {
    return tone(color)(theme) === "dark";
  };
};
/**
 * Determines if a color tone is "light"
 * @param color - the color in hex, rgb, or hsl
 */

var isLight = function isLight(color) {
  return function (theme) {
    return tone(color)(theme) === "light";
  };
};
/**
 * Make a color transparent
 * @param color - the color in hex, rgb, or hsl
 * @param opacity - the amount of opacity the color should have (0-1)
 */

var transparentize = function transparentize(color, opacity) {
  return function (theme) {
    var raw = getColor(theme, color);
    return new TinyColor(raw).setAlpha(opacity).toRgbString();
  };
};
/**
 * Add white to a color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount white to add (0-100)
 */

var whiten = function whiten(color, amount) {
  return function (theme) {
    var raw = getColor(theme, color);
    return new TinyColor(raw).mix("#fff", amount).toHexString();
  };
};
/**
 * Add black to a color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount black to add (0-100)
 */

var blacken = function blacken(color, amount) {
  return function (theme) {
    var raw = getColor(theme, color);
    return new TinyColor(raw).mix("#000", amount).toHexString();
  };
};
/**
 * Darken a specified color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount to darken (0-100)
 */

var darken = function darken(color, amount) {
  return function (theme) {
    var raw = getColor(theme, color);
    return new TinyColor(raw).darken(amount).toHexString();
  };
};
/**
 * Lighten a specified color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount to lighten (0-100)
 */

var lighten = function lighten(color, amount) {
  return function (theme) {
    return new TinyColor(getColor(theme, color)).lighten(amount).toHexString();
  };
};
/**
 * Checks the contract ratio of between 2 colors,
 * based on the Web Content Accessibility Guidelines (Version 2.0).
 *
 * @param fg - the foreground or text color
 * @param bg - the background color
 */

var contrast = function contrast(fg, bg) {
  return function (theme) {
    return readability(getColor(theme, bg), getColor(theme, fg));
  };
};
/**
 * Checks if a color meets the Web Content Accessibility
 * Guidelines (Version 2.0) for constract ratio.
 *
 * @param fg - the foreground or text color
 * @param bg - the background color
 */

var isAccessible = function isAccessible(textColor, bgColor, options) {
  return function (theme) {
    return isReadable(getColor(theme, bgColor), getColor(theme, textColor), options);
  };
};
var complementary = function complementary(color) {
  return function (theme) {
    return new TinyColor(getColor(theme, color)).complement().toHexString();
  };
};
function generateStripe(size, color) {
  if (size === void 0) {
    size = "1rem";
  }

  if (color === void 0) {
    color = "rgba(255, 255, 255, 0.15)";
  }

  return {
    backgroundImage: "linear-gradient(\n    45deg,\n    " + color + " 25%,\n    transparent 25%,\n    transparent 50%,\n    " + color + " 50%,\n    " + color + " 75%,\n    transparent 75%,\n    transparent\n  )",
    backgroundSize: size + " " + size
  };
}
function randomColor(opts) {
  var fallback = random().toHexString();

  if (!opts || isEmptyObject(opts)) {
    return fallback;
  }

  if (opts.string && opts.colors) {
    return randomColorFromList(opts.string, opts.colors);
  }

  if (opts.string && !opts.colors) {
    return randomColorFromString(opts.string);
  }

  if (opts.colors && !opts.string) {
    return randomFromList(opts.colors);
  }

  return fallback;
}

function randomColorFromString(str) {
  var hash = 0;
  if (str.length === 0) return hash.toString();

  for (var i = 0; i < str.length; i += 1) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
    hash = hash & hash;
  }

  var color = "#";

  for (var j = 0; j < 3; j += 1) {
    var value = hash >> j * 8 & 255;
    color += ("00" + value.toString(16)).substr(-2);
  }

  return color;
}

function randomColorFromList(str, list) {
  var index = 0;
  if (str.length === 0) return list[0];

  for (var i = 0; i < str.length; i += 1) {
    index = str.charCodeAt(i) + ((index << 5) - index);
    index = index & index;
  }

  index = (index % list.length + list.length) % list.length;
  return list[index];
}

function randomFromList(list) {
  return list[Math.floor(Math.random() * list.length)];
}

function mode(light, dark) {
  return function (props) {
    return props.colorMode === "dark" ? dark : light;
  };
}
function orient(options) {
  var orientation = options.orientation,
      vertical = options.vertical,
      horizontal = options.horizontal;
  if (!orientation) return {};
  return orientation === "vertical" ? vertical : horizontal;
}

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

var createBreakpoints = function createBreakpoints(config) {
  warn({
    condition: true,
    message: ["[chakra-ui]: createBreakpoints(...) will be deprecated pretty soon", "simply pass the breakpoints as an object. Remove the createBreakpoint(..) call"].join("")
  });
  return _extends({
    base: "0em"
  }, config);
};

function _defineProperties(target, props) {
  for (var i = 0; i < props.length; i++) {
    var descriptor = props[i];
    descriptor.enumerable = descriptor.enumerable || false;
    descriptor.configurable = true;
    if ("value" in descriptor) descriptor.writable = true;
    Object.defineProperty(target, descriptor.key, descriptor);
  }
}

function _createClass(Constructor, protoProps, staticProps) {
  if (protoProps) _defineProperties(Constructor.prototype, protoProps);
  if (staticProps) _defineProperties(Constructor, staticProps);
  return Constructor;
}

/**
 * Used to define the anatomy/parts of a component in a way that provides
 * a consistent API for `className`, css selector and `theming`.
 */

var Anatomy = /*#__PURE__*/function () {
  function Anatomy(name) {
    var _this = this;

    this.map = {};
    this.called = false;

    this.assert = function () {
      if (!_this.called) {
        _this.called = true;
        return;
      }

      throw new Error("[anatomy] .part(...) should only be called once. Did you mean to use .extend(...) ?");
    };

    this.parts = function () {
      _this.assert();

      for (var _len = arguments.length, values = new Array(_len), _key = 0; _key < _len; _key++) {
        values[_key] = arguments[_key];
      }

      for (var _i = 0, _values = values; _i < _values.length; _i++) {
        var part = _values[_i];
        _this.map[part] = _this.toPart(part);
      }

      return _this;
    };

    this.extend = function () {
      for (var _len2 = arguments.length, parts = new Array(_len2), _key2 = 0; _key2 < _len2; _key2++) {
        parts[_key2] = arguments[_key2];
      }

      for (var _i2 = 0, _parts = parts; _i2 < _parts.length; _i2++) {
        var part = _parts[_i2];
        if (part in _this.map) continue;
        _this.map[part] = _this.toPart(part);
      }

      return _this;
    };

    this.toPart = function (part) {
      var el = ["container", "root"].includes(part != null ? part : "") ? [_this.name] : [_this.name, part];
      var attr = el.filter(Boolean).join("__");
      var className = "chakra-" + attr;
      var partObj = {
        className: className,
        selector: "." + className,
        toString: function toString() {
          return part;
        }
      };
      return partObj;
    };

    this.__type = {};
  }
  /**
   * Prevents user from calling `.parts` multiple times.
   * It should only be called once.
   */


  _createClass(Anatomy, [{
    key: "selectors",
    get:
    /**
     * Get all selectors for the component anatomy
     */
    function get() {
      var value = fromEntries(Object.entries(this.map).map(function (_ref) {
        var key = _ref[0],
            part = _ref[1];
        return [key, part.selector];
      }));
      return value;
    }
    /**
     * Get all classNames for the component anatomy
     */

  }, {
    key: "classNames",
    get: function get() {
      var value = fromEntries(Object.entries(this.map).map(function (_ref2) {
        var key = _ref2[0],
            part = _ref2[1];
        return [key, part.className];
      }));
      return value;
    }
    /**
     * Get all parts as array of string
     */

  }, {
    key: "keys",
    get: function get() {
      return Object.keys(this.map);
    }
    /**
     * Creates the part object for the given part
     */

  }]);

  return Anatomy;
}();
function anatomy(name) {
  return new Anatomy(name);
}

function toRef(operand) {
  if (isObject(operand) && operand.reference) {
    return operand.reference;
  }

  return String(operand);
}

var toExpr = function toExpr(operator) {
  for (var _len = arguments.length, operands = new Array(_len > 1 ? _len - 1 : 0), _key = 1; _key < _len; _key++) {
    operands[_key - 1] = arguments[_key];
  }

  return operands.map(toRef).join(" " + operator + " ").replace(/calc/g, "");
};

var _add = function add() {
  for (var _len2 = arguments.length, operands = new Array(_len2), _key2 = 0; _key2 < _len2; _key2++) {
    operands[_key2] = arguments[_key2];
  }

  return "calc(" + toExpr.apply(void 0, ["+"].concat(operands)) + ")";
};

var _subtract = function subtract() {
  for (var _len3 = arguments.length, operands = new Array(_len3), _key3 = 0; _key3 < _len3; _key3++) {
    operands[_key3] = arguments[_key3];
  }

  return "calc(" + toExpr.apply(void 0, ["-"].concat(operands)) + ")";
};

var _multiply = function multiply() {
  for (var _len4 = arguments.length, operands = new Array(_len4), _key4 = 0; _key4 < _len4; _key4++) {
    operands[_key4] = arguments[_key4];
  }

  return "calc(" + toExpr.apply(void 0, ["*"].concat(operands)) + ")";
};

var _divide = function divide() {
  for (var _len5 = arguments.length, operands = new Array(_len5), _key5 = 0; _key5 < _len5; _key5++) {
    operands[_key5] = arguments[_key5];
  }

  return "calc(" + toExpr.apply(void 0, ["/"].concat(operands)) + ")";
};

var _negate = function negate(x) {
  var value = toRef(x);

  if (value != null && !Number.isNaN(parseFloat(value))) {
    return String(value).startsWith("-") ? String(value).slice(1) : "-" + value;
  }

  return _multiply(value, -1);
};

var calc = Object.assign(function (x) {
  return {
    add: function add() {
      for (var _len6 = arguments.length, operands = new Array(_len6), _key6 = 0; _key6 < _len6; _key6++) {
        operands[_key6] = arguments[_key6];
      }

      return calc(_add.apply(void 0, [x].concat(operands)));
    },
    subtract: function subtract() {
      for (var _len7 = arguments.length, operands = new Array(_len7), _key7 = 0; _key7 < _len7; _key7++) {
        operands[_key7] = arguments[_key7];
      }

      return calc(_subtract.apply(void 0, [x].concat(operands)));
    },
    multiply: function multiply() {
      for (var _len8 = arguments.length, operands = new Array(_len8), _key8 = 0; _key8 < _len8; _key8++) {
        operands[_key8] = arguments[_key8];
      }

      return calc(_multiply.apply(void 0, [x].concat(operands)));
    },
    divide: function divide() {
      for (var _len9 = arguments.length, operands = new Array(_len9), _key9 = 0; _key9 < _len9; _key9++) {
        operands[_key9] = arguments[_key9];
      }

      return calc(_divide.apply(void 0, [x].concat(operands)));
    },
    negate: function negate() {
      return calc(_negate(x));
    },
    toString: function toString() {
      return x.toString();
    }
  };
}, {
  add: _add,
  subtract: _subtract,
  multiply: _multiply,
  divide: _divide,
  negate: _negate
});

function isDecimal(value) {
  return !Number.isInteger(parseFloat(value.toString()));
}

function replaceWhiteSpace(value, replaceValue) {
  if (replaceValue === void 0) {
    replaceValue = "-";
  }

  return value.replace(/\s+/g, replaceValue);
}

function escape(value) {
  var valueStr = replaceWhiteSpace(value.toString());
  if (valueStr.includes("\\.")) return value;
  return isDecimal(value) ? valueStr.replace(".", "\\.") : value;
}

function addPrefix(value, prefix) {
  if (prefix === void 0) {
    prefix = "";
  }

  return [prefix, escape(value)].filter(Boolean).join("-");
}
function toVarRef(name, fallback) {
  return "var(" + escape(name) + (fallback ? ", " + fallback : "") + ")";
}
function toVar(value, prefix) {
  if (prefix === void 0) {
    prefix = "";
  }

  return "--" + addPrefix(value, prefix);
}
function cssVar(name, options) {
  var cssVariable = toVar(name, options == null ? void 0 : options.prefix);
  return {
    variable: cssVariable,
    reference: toVarRef(cssVariable, getFallback(options == null ? void 0 : options.fallback))
  };
}

function getFallback(fallback) {
  if (typeof fallback === "string") return fallback;
  return fallback == null ? void 0 : fallback.reference;
}

export { Anatomy, addPrefix, anatomy, blacken, calc, complementary, contrast, createBreakpoints, cssVar, darken, generateStripe, getColor, isAccessible, isDark, isDecimal, isLight, lighten, mode, orient, randomColor, toVar, toVarRef, tone, transparentize, whiten };
