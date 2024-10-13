import { forwardRef, chakra, useStyleConfig, omitThemingProps, useMultiStyleConfig, StylesProvider, useStyles, tokenToCSSVar } from '@chakra-ui/system';
import { cx, mapResponsive, __DEV__, filterUndefined, isNull, isNumber } from '@chakra-ui/utils';
import * as React from 'react';
import { Icon } from '@chakra-ui/icon';
import { getValidChildren } from '@chakra-ui/react-utils';

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

var _excluded$h = ["ratio", "children", "className"];

/**
 * React component used to cropping media (videos, images and maps)
 * to a desired aspect ratio.
 *
 * @see Docs https://chakra-ui.com/aspectratiobox
 */
var AspectRatio = /*#__PURE__*/forwardRef(function (props, ref) {
  var _props$ratio = props.ratio,
      ratio = _props$ratio === void 0 ? 4 / 3 : _props$ratio,
      children = props.children,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded$h); // enforce single child


  var child = React.Children.only(children);

  var _className = cx("chakra-aspect-ratio", className);

  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    position: "relative",
    className: _className,
    _before: {
      height: 0,
      content: "\"\"",
      display: "block",
      paddingBottom: mapResponsive(ratio, function (r) {
        return 1 / r * 100 + "%";
      })
    },
    __css: {
      "& > *:not(style)": {
        overflow: "hidden",
        position: "absolute",
        top: "0",
        right: "0",
        bottom: "0",
        left: "0",
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        width: "100%",
        height: "100%"
      },
      "& > img, & > video": {
        objectFit: "cover"
      }
    }
  }, rest), child);
});

if (__DEV__) {
  AspectRatio.displayName = "AspectRatio";
}

var _excluded$g = ["className"];

/**
 * React component used to display notifications, messages, or
 * statuses in different shapes and sizes.
 *
 * @see Docs https://chakra-ui.com/badge
 */
var Badge = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Badge", props);

  var _omitThemingProps = omitThemingProps(props);
      _omitThemingProps.className;
      var rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$g);

  return /*#__PURE__*/React.createElement(chakra.span, _extends({
    ref: ref,
    className: cx("chakra-badge", props.className)
  }, rest, {
    __css: _extends({
      display: "inline-block",
      whiteSpace: "nowrap",
      verticalAlign: "middle"
    }, styles)
  }));
});

if (__DEV__) {
  Badge.displayName = "Badge";
}

var _excluded$f = ["size", "centerContent"],
    _excluded2$5 = ["size"];

/**
 * Box is the most abstract component on top of which other chakra
 * components are built. It renders a `div` element by default.
 *
 * @see Docs https://chakra-ui.com/box
 */
var Box = chakra("div");

if (__DEV__) {
  Box.displayName = "Box";
}
/**
 * As a constraint, you can't pass size related props
 * Only `size` would be allowed
 */


var Square = /*#__PURE__*/forwardRef(function (props, ref) {
  var size = props.size,
      _props$centerContent = props.centerContent,
      centerContent = _props$centerContent === void 0 ? true : _props$centerContent,
      rest = _objectWithoutPropertiesLoose(props, _excluded$f);

  var styles = centerContent ? {
    display: "flex",
    alignItems: "center",
    justifyContent: "center"
  } : {};
  return /*#__PURE__*/React.createElement(Box, _extends({
    ref: ref,
    boxSize: size,
    __css: _extends({}, styles, {
      flexShrink: 0,
      flexGrow: 0
    })
  }, rest));
});

if (__DEV__) {
  Square.displayName = "Square";
}

var Circle = /*#__PURE__*/forwardRef(function (props, ref) {
  var size = props.size,
      rest = _objectWithoutPropertiesLoose(props, _excluded2$5);

  return /*#__PURE__*/React.createElement(Square, _extends({
    size: size,
    ref: ref,
    borderRadius: "9999px"
  }, rest));
});

if (__DEV__) {
  Circle.displayName = "Circle";
}

var _excluded$e = ["axis"];

/**
 * React component used to horizontally and vertically center its child.
 * It uses the popular `display: flex` centering technique.
 *
 * @see Docs https://chakra-ui.com/center
 */
var Center = chakra("div", {
  baseStyle: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center"
  }
});

if (__DEV__) {
  Center.displayName = "Center";
}

var centerStyles = {
  horizontal: {
    insetStart: "50%",
    transform: "translateX(-50%)"
  },
  vertical: {
    top: "50%",
    transform: "translateY(-50%)"
  },
  both: {
    insetStart: "50%",
    top: "50%",
    transform: "translate(-50%, -50%)"
  }
};
/**
 * React component used to horizontally and vertically center an element
 * relative to its parent dimensions.
 *
 * It uses the `position: absolute` strategy.
 *
 * @see Docs https://chakra-ui.com/center
 * @see WebDev https://web.dev/centering-in-css/#5.-pop-and-plop
 */

var AbsoluteCenter = /*#__PURE__*/forwardRef(function (props, ref) {
  var _props$axis = props.axis,
      axis = _props$axis === void 0 ? "both" : _props$axis,
      rest = _objectWithoutPropertiesLoose(props, _excluded$e);

  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    __css: centerStyles[axis]
  }, rest, {
    position: "absolute"
  }));
});

var _excluded$d = ["className"];

/**
 * React component to render inline code snippets.
 *
 * @see Docs https://chakra-ui.com/code
 */
var Code = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Code", props);

  var _omitThemingProps = omitThemingProps(props);
      _omitThemingProps.className;
      var rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$d);

  return /*#__PURE__*/React.createElement(chakra.code, _extends({
    ref: ref,
    className: cx("chakra-code", props.className)
  }, rest, {
    __css: _extends({
      display: "inline-block"
    }, styles)
  }));
});

if (__DEV__) {
  Code.displayName = "Code";
}

var _excluded$c = ["className", "centerContent"];

/**
 * Layout component used to wrap app or website content
 *
 * It sets `margin-left` and `margin-right` to `auto`,
 * to keep its content centered.
 *
 * It also sets a default max-width of `60ch` (60 characters).
 */
var Container = /*#__PURE__*/forwardRef(function (props, ref) {
  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      centerContent = _omitThemingProps.centerContent,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$c);

  var styles = useStyleConfig("Container", props);
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    className: cx("chakra-container", className)
  }, rest, {
    __css: _extends({}, styles, centerContent && {
      display: "flex",
      flexDirection: "column",
      alignItems: "center"
    })
  }));
});

if (__DEV__) {
  Container.displayName = "Container";
}

var _excluded$b = ["borderLeftWidth", "borderBottomWidth", "borderTopWidth", "borderRightWidth", "borderWidth", "borderStyle", "borderColor"],
    _excluded2$4 = ["className", "orientation", "__css"];
/**
 * Layout component used to visually separate content in a list or group.
 * It display a thin horizontal or vertical line, and renders a `hr` tag.
 *
 * @see Docs https://chakra-ui.com/divider
 */

var Divider = /*#__PURE__*/forwardRef(function (props, ref) {
  var _useStyleConfig = useStyleConfig("Divider", props),
      borderLeftWidth = _useStyleConfig.borderLeftWidth,
      borderBottomWidth = _useStyleConfig.borderBottomWidth,
      borderTopWidth = _useStyleConfig.borderTopWidth,
      borderRightWidth = _useStyleConfig.borderRightWidth,
      borderWidth = _useStyleConfig.borderWidth,
      borderStyle = _useStyleConfig.borderStyle,
      borderColor = _useStyleConfig.borderColor,
      styles = _objectWithoutPropertiesLoose(_useStyleConfig, _excluded$b);

  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      _omitThemingProps$ori = _omitThemingProps.orientation,
      orientation = _omitThemingProps$ori === void 0 ? "horizontal" : _omitThemingProps$ori,
      __css = _omitThemingProps.__css,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded2$4);

  var dividerStyles = {
    vertical: {
      borderLeftWidth: borderLeftWidth || borderRightWidth || borderWidth || "1px",
      height: "100%"
    },
    horizontal: {
      borderBottomWidth: borderBottomWidth || borderTopWidth || borderWidth || "1px",
      width: "100%"
    }
  };
  return /*#__PURE__*/React.createElement(chakra.hr, _extends({
    ref: ref,
    "aria-orientation": orientation
  }, rest, {
    __css: _extends({}, styles, {
      border: "0",
      borderColor: borderColor,
      borderStyle: borderStyle
    }, dividerStyles[orientation], __css),
    className: cx("chakra-divider", className)
  }));
});

if (__DEV__) {
  Divider.displayName = "Divider";
}

var _excluded$a = ["direction", "align", "justify", "wrap", "basis", "grow", "shrink"];

/**
 * React component used to create flexbox layouts.
 *
 * It renders a `div` with `display: flex` and
 * comes with helpful style shorthand.
 *
 * @see Docs https://chakra-ui.com/flex
 */
var Flex = /*#__PURE__*/forwardRef(function (props, ref) {
  var direction = props.direction,
      align = props.align,
      justify = props.justify,
      wrap = props.wrap,
      basis = props.basis,
      grow = props.grow,
      shrink = props.shrink,
      rest = _objectWithoutPropertiesLoose(props, _excluded$a);

  var styles = {
    display: "flex",
    flexDirection: direction,
    alignItems: align,
    justifyContent: justify,
    flexWrap: wrap,
    flexBasis: basis,
    flexGrow: grow,
    flexShrink: shrink
  };
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    __css: styles
  }, rest));
});

if (__DEV__) {
  Flex.displayName = "Flex";
}

var _excluded$9 = ["area", "templateAreas", "gap", "rowGap", "columnGap", "column", "row", "autoFlow", "autoRows", "templateRows", "autoColumns", "templateColumns"],
    _excluded2$3 = ["colSpan", "colStart", "colEnd", "rowEnd", "rowSpan", "rowStart"];

/**
 * React component used to create grid layouts.
 *
 * It renders a `div` with `display: grid` and
 * comes with helpful style shorthand.
 *
 * @see Docs https://chakra-ui.com/grid
 */
var Grid = /*#__PURE__*/forwardRef(function (props, ref) {
  var area = props.area,
      templateAreas = props.templateAreas,
      gap = props.gap,
      rowGap = props.rowGap,
      columnGap = props.columnGap,
      column = props.column,
      row = props.row,
      autoFlow = props.autoFlow,
      autoRows = props.autoRows,
      templateRows = props.templateRows,
      autoColumns = props.autoColumns,
      templateColumns = props.templateColumns,
      rest = _objectWithoutPropertiesLoose(props, _excluded$9);

  var styles = {
    display: "grid",
    gridArea: area,
    gridTemplateAreas: templateAreas,
    gridGap: gap,
    gridRowGap: rowGap,
    gridColumnGap: columnGap,
    gridAutoColumns: autoColumns,
    gridColumn: column,
    gridRow: row,
    gridAutoFlow: autoFlow,
    gridAutoRows: autoRows,
    gridTemplateRows: templateRows,
    gridTemplateColumns: templateColumns
  };
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    __css: styles
  }, rest));
});

if (__DEV__) {
  Grid.displayName = "Grid";
}

function spanFn(span) {
  return mapResponsive(span, function (value) {
    return value === "auto" ? "auto" : "span " + value + "/span " + value;
  });
}

var GridItem = /*#__PURE__*/forwardRef(function (props, ref) {
  var colSpan = props.colSpan,
      colStart = props.colStart,
      colEnd = props.colEnd,
      rowEnd = props.rowEnd,
      rowSpan = props.rowSpan,
      rowStart = props.rowStart,
      rest = _objectWithoutPropertiesLoose(props, _excluded2$3);

  var styles = filterUndefined({
    gridColumn: spanFn(colSpan),
    gridRow: spanFn(rowSpan),
    gridColumnStart: colStart,
    gridColumnEnd: colEnd,
    gridRowStart: rowStart,
    gridRowEnd: rowEnd
  });
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    __css: styles
  }, rest));
});

var _excluded$8 = ["className"];
var Heading = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Heading", props);

  var _omitThemingProps = omitThemingProps(props);
      _omitThemingProps.className;
      var rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$8);

  return /*#__PURE__*/React.createElement(chakra.h2, _extends({
    ref: ref,
    className: cx("chakra-heading", props.className)
  }, rest, {
    __css: styles
  }));
});

if (__DEV__) {
  Heading.displayName = "Heading";
}

var _excluded$7 = ["className"];

/**
 * Semantic component to render a keyboard shortcut
 * within an application.
 *
 * @example
 *
 * ```jsx
 * <Kbd>⌘ + T</Kbd>
 * ```
 *
 * @see Docs https://chakra-ui.com/kbd
 */
var Kbd = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Kbd", props);

  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$7);

  return /*#__PURE__*/React.createElement(chakra.kbd, _extends({
    ref: ref,
    className: cx("chakra-kbd", className)
  }, rest, {
    __css: _extends({
      fontFamily: "mono"
    }, styles)
  }));
});

if (__DEV__) {
  Kbd.displayName = "Kbd";
}

var _excluded$6 = ["className", "isExternal"];

/**
 * Links are accessible elements used primarily for navigation.
 *
 * It integrates well with other routing libraries like
 * React Router, Reach Router and Next.js Link.
 *
 * @example
 *
 * ```jsx
 * <Link as={ReactRouterLink} to="/home">Home</Link>
 * ```
 *
 * @see Docs https://chakra-ui.com/link
 */
var Link = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Link", props);

  var _omitThemingProps = omitThemingProps(props),
      className = _omitThemingProps.className,
      isExternal = _omitThemingProps.isExternal,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$6);

  return /*#__PURE__*/React.createElement(chakra.a, _extends({
    target: isExternal ? "_blank" : undefined,
    rel: isExternal ? "noopener noreferrer" : undefined,
    ref: ref,
    className: cx("chakra-link", className)
  }, rest, {
    __css: styles
  }));
});

if (__DEV__) {
  Link.displayName = "Link";
}

var _excluded$5 = ["children", "styleType", "stylePosition", "spacing"],
    _excluded2$2 = ["as"],
    _excluded3 = ["as"];

/**
 * List is used to display list items, it renders a `<ul>` by default.
 *
 * @see Docs https://chakra-ui.com/list
 */
var List = /*#__PURE__*/forwardRef(function (props, ref) {
  var _ref;

  var styles = useMultiStyleConfig("List", props);

  var _omitThemingProps = omitThemingProps(props),
      children = _omitThemingProps.children,
      _omitThemingProps$sty = _omitThemingProps.styleType,
      styleType = _omitThemingProps$sty === void 0 ? "none" : _omitThemingProps$sty,
      stylePosition = _omitThemingProps.stylePosition,
      spacing = _omitThemingProps.spacing,
      rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$5);

  var validChildren = getValidChildren(children);
  var selector = "& > *:not(style) ~ *:not(style)";
  var spacingStyle = spacing ? (_ref = {}, _ref[selector] = {
    mt: spacing
  }, _ref) : {};
  return /*#__PURE__*/React.createElement(StylesProvider, {
    value: styles
  }, /*#__PURE__*/React.createElement(chakra.ul, _extends({
    ref: ref,
    listStyleType: styleType,
    listStylePosition: stylePosition
    /**
     * We added this role to fix the Safari accessibility issue with list-style-type: none
     * @see https://www.scottohara.me/blog/2019/01/12/lists-and-safari.html
     */
    ,
    role: "list",
    __css: _extends({}, styles.container, spacingStyle)
  }, rest), validChildren));
});

if (__DEV__) {
  List.displayName = "List";
}

var OrderedList = /*#__PURE__*/forwardRef(function (props, ref) {
  props.as;
      var rest = _objectWithoutPropertiesLoose(props, _excluded2$2);

  return /*#__PURE__*/React.createElement(List, _extends({
    ref: ref,
    as: "ol",
    styleType: "decimal",
    marginStart: "1em"
  }, rest));
});

if (__DEV__) {
  OrderedList.displayName = "OrderedList";
}

var UnorderedList = /*#__PURE__*/forwardRef(function (props, ref) {
  props.as;
      var rest = _objectWithoutPropertiesLoose(props, _excluded3);

  return /*#__PURE__*/React.createElement(List, _extends({
    ref: ref,
    as: "ul",
    styleType: "initial",
    marginStart: "1em"
  }, rest));
});

if (__DEV__) {
  UnorderedList.displayName = "UnorderedList";
}

/**
 * ListItem
 *
 * Used to render a list item
 */
var ListItem = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(chakra.li, _extends({
    ref: ref
  }, props, {
    __css: styles.item
  }));
});

if (__DEV__) {
  ListItem.displayName = "ListItem";
}
/**
 * ListIcon
 *
 * Used to render an icon beside the list item text
 */


var ListIcon = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyles();
  return /*#__PURE__*/React.createElement(Icon, _extends({
    ref: ref,
    role: "presentation"
  }, props, {
    __css: styles.icon
  }));
});

if (__DEV__) {
  ListIcon.displayName = "ListIcon";
}

var _excluded$4 = ["columns", "spacingX", "spacingY", "spacing", "minChildWidth"];

/**
 * SimpleGrid
 *
 * React component make that providers a simpler interface, and
 * make its easy to create responsive grid layouts.
 *
 * @see Docs https://chakra-ui.com/simplegrid
 */
var SimpleGrid = /*#__PURE__*/forwardRef(function (props, ref) {
  var columns = props.columns,
      spacingX = props.spacingX,
      spacingY = props.spacingY,
      spacing = props.spacing,
      minChildWidth = props.minChildWidth,
      rest = _objectWithoutPropertiesLoose(props, _excluded$4);

  var templateColumns = minChildWidth ? widthToColumns(minChildWidth) : countToColumns(columns);
  return /*#__PURE__*/React.createElement(Grid, _extends({
    ref: ref,
    gap: spacing,
    columnGap: spacingX,
    rowGap: spacingY,
    templateColumns: templateColumns
  }, rest));
});

if (__DEV__) {
  SimpleGrid.displayName = "SimpleGrid";
}

function toPx(n) {
  return isNumber(n) ? n + "px" : n;
}

function widthToColumns(width) {
  return mapResponsive(width, function (value) {
    return isNull(value) ? null : "repeat(auto-fit, minmax(" + toPx(value) + ", 1fr))";
  });
}

function countToColumns(count) {
  return mapResponsive(count, function (value) {
    return isNull(value) ? null : "repeat(" + value + ", minmax(0, 1fr))";
  });
}

/**
 * A flexible flex spacer that expands along the major axis of its containing flex layout.
 * It renders a `div` by default, and takes up any available space.
 *
 * @see Docs https://chakra-ui.com/flex#using-the-spacer
 */
var Spacer = chakra("div", {
  baseStyle: {
    flex: 1,
    justifySelf: "stretch",
    alignSelf: "stretch"
  }
});

if (__DEV__) {
  Spacer.displayName = "Spacer";
}

/**
 * If we ever run into SSR issues with this, check this post to find a fix for it:
 * @see https://medium.com/@emmenko/patching-lobotomized-owl-selector-for-emotion-ssr-5a582a3c424c
 */
var selector = "& > *:not(style) ~ *:not(style)";
function getStackStyles(options) {
  var _ref;

  var spacing = options.spacing,
      direction = options.direction;
  var directionStyles = {
    column: {
      marginTop: spacing,
      marginEnd: 0,
      marginBottom: 0,
      marginStart: 0
    },
    row: {
      marginTop: 0,
      marginEnd: 0,
      marginBottom: 0,
      marginStart: spacing
    },
    "column-reverse": {
      marginTop: 0,
      marginEnd: 0,
      marginBottom: spacing,
      marginStart: 0
    },
    "row-reverse": {
      marginTop: 0,
      marginEnd: spacing,
      marginBottom: 0,
      marginStart: 0
    }
  };
  return _ref = {
    flexDirection: direction
  }, _ref[selector] = mapResponsive(direction, function (value) {
    return directionStyles[value];
  }), _ref;
}
function getDividerStyles(options) {
  var spacing = options.spacing,
      direction = options.direction;
  var dividerStyles = {
    column: {
      my: spacing,
      mx: 0,
      borderLeftWidth: 0,
      borderBottomWidth: "1px"
    },
    "column-reverse": {
      my: spacing,
      mx: 0,
      borderLeftWidth: 0,
      borderBottomWidth: "1px"
    },
    row: {
      mx: spacing,
      my: 0,
      borderLeftWidth: "1px",
      borderBottomWidth: 0
    },
    "row-reverse": {
      mx: spacing,
      my: 0,
      borderLeftWidth: "1px",
      borderBottomWidth: 0
    }
  };
  return {
    "&": mapResponsive(direction, function (value) {
      return dividerStyles[value];
    })
  };
}

var _excluded$3 = ["isInline", "direction", "align", "justify", "spacing", "wrap", "children", "divider", "className", "shouldWrapChildren"];
var StackDivider = function StackDivider(props) {
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    className: "chakra-stack__divider"
  }, props, {
    __css: _extends({}, props["__css"], {
      borderWidth: 0,
      alignSelf: "stretch",
      borderColor: "inherit",
      width: "auto",
      height: "auto"
    })
  }));
};
var StackItem = function StackItem(props) {
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    className: "chakra-stack__item"
  }, props, {
    __css: _extends({
      display: "inline-block",
      flex: "0 0 auto",
      minWidth: 0
    }, props["__css"])
  }));
};

/**
 * Stacks help you easily create flexible and automatically distributed layouts
 *
 * You can stack elements in the horizontal or vertical direction,
 * and apply a space or/and divider between each element.
 *
 * It uses `display: flex` internally and renders a `div`.
 *
 * @see Docs https://chakra-ui.com/stack
 *
 */
var Stack = /*#__PURE__*/forwardRef(function (props, ref) {
  var _ref;

  var isInline = props.isInline,
      directionProp = props.direction,
      align = props.align,
      justify = props.justify,
      _props$spacing = props.spacing,
      spacing = _props$spacing === void 0 ? "0.5rem" : _props$spacing,
      wrap = props.wrap,
      children = props.children,
      divider = props.divider,
      className = props.className,
      shouldWrapChildren = props.shouldWrapChildren,
      rest = _objectWithoutPropertiesLoose(props, _excluded$3);

  var direction = isInline ? "row" : directionProp != null ? directionProp : "column";
  var styles = React.useMemo(function () {
    return getStackStyles({
      direction: direction,
      spacing: spacing
    });
  }, [direction, spacing]);
  var dividerStyle = React.useMemo(function () {
    return getDividerStyles({
      spacing: spacing,
      direction: direction
    });
  }, [spacing, direction]);
  var hasDivider = !!divider;
  var shouldUseChildren = !shouldWrapChildren && !hasDivider;
  var validChildren = getValidChildren(children);
  var clones = shouldUseChildren ? validChildren : validChildren.map(function (child, index) {
    // Prefer provided child key, fallback to index
    var key = typeof child.key !== "undefined" ? child.key : index;
    var isLast = index + 1 === validChildren.length;
    var wrappedChild = /*#__PURE__*/React.createElement(StackItem, {
      key: key
    }, child);

    var _child = shouldWrapChildren ? wrappedChild : child;

    if (!hasDivider) return _child;
    var clonedDivider = /*#__PURE__*/React.cloneElement(divider, {
      __css: dividerStyle
    });

    var _divider = isLast ? null : clonedDivider;

    return /*#__PURE__*/React.createElement(React.Fragment, {
      key: key
    }, _child, _divider);
  });

  var _className = cx("chakra-stack", className);

  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    display: "flex",
    alignItems: align,
    justifyContent: justify,
    flexDirection: styles.flexDirection,
    flexWrap: wrap,
    className: _className,
    __css: hasDivider ? {} : (_ref = {}, _ref[selector] = styles[selector], _ref)
  }, rest), clones);
});

if (__DEV__) {
  Stack.displayName = "Stack";
}
/**
 * A view that arranges its children in a horizontal line.
 */


var HStack = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(Stack, _extends({
    align: "center"
  }, props, {
    direction: "row",
    ref: ref
  }));
});

if (__DEV__) {
  HStack.displayName = "HStack";
}
/**
 * A view that arranges its children in a vertical line.
 */


var VStack = /*#__PURE__*/forwardRef(function (props, ref) {
  return /*#__PURE__*/React.createElement(Stack, _extends({
    align: "center"
  }, props, {
    direction: "column",
    ref: ref
  }));
});

if (__DEV__) {
  VStack.displayName = "VStack";
}

var _excluded$2 = ["className", "align", "decoration", "casing"];

/**
 * Used to render texts or paragraphs.
 *
 * @see Docs https://chakra-ui.com/text
 */
var Text = /*#__PURE__*/forwardRef(function (props, ref) {
  var styles = useStyleConfig("Text", props);

  var _omitThemingProps = omitThemingProps(props);
      _omitThemingProps.className;
      _omitThemingProps.align;
      _omitThemingProps.decoration;
      _omitThemingProps.casing;
      var rest = _objectWithoutPropertiesLoose(_omitThemingProps, _excluded$2);

  var aliasedProps = filterUndefined({
    textAlign: props.align,
    textDecoration: props.decoration,
    textTransform: props.casing
  });
  return /*#__PURE__*/React.createElement(chakra.p, _extends({
    ref: ref,
    className: cx("chakra-text", props.className)
  }, aliasedProps, rest, {
    __css: styles
  }));
});

if (__DEV__) {
  Text.displayName = "Text";
}

var _excluded$1 = ["spacing", "children", "justify", "direction", "align", "className", "shouldWrapChildren"],
    _excluded2$1 = ["className"];

/**
 * Layout component used to stack elements that differ in length
 * and are liable to wrap.
 *
 * Common use cases:
 * - Buttons that appear together at the end of forms
 * - Lists of tags and chips
 *
 * @see Docs https://chakra-ui.com/wrap
 */
var Wrap = /*#__PURE__*/forwardRef(function (props, ref) {
  var _props$spacing = props.spacing,
      spacing = _props$spacing === void 0 ? "0.5rem" : _props$spacing,
      children = props.children,
      justify = props.justify,
      direction = props.direction,
      align = props.align,
      className = props.className,
      shouldWrapChildren = props.shouldWrapChildren,
      rest = _objectWithoutPropertiesLoose(props, _excluded$1);

  var styles = React.useMemo(function () {
    return {
      "--chakra-wrap-spacing": function chakraWrapSpacing(theme) {
        return mapResponsive(spacing, function (value) {
          return tokenToCSSVar("space", value)(theme);
        });
      },
      "--wrap-spacing": "calc(var(--chakra-wrap-spacing) / 2)",
      display: "flex",
      flexWrap: "wrap",
      justifyContent: justify,
      alignItems: align,
      flexDirection: direction,
      listStyleType: "none",
      padding: "0",
      margin: "calc(var(--wrap-spacing) * -1)",
      "& > *:not(style)": {
        margin: "var(--wrap-spacing)"
      }
    };
  }, [spacing, justify, align, direction]);
  var childrenToRender = shouldWrapChildren ? React.Children.map(children, function (child, index) {
    return /*#__PURE__*/React.createElement(WrapItem, {
      key: index
    }, child);
  }) : children;
  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    className: cx("chakra-wrap", className)
  }, rest), /*#__PURE__*/React.createElement(chakra.ul, {
    className: "chakra-wrap__list",
    __css: styles
  }, childrenToRender));
});

if (__DEV__) {
  Wrap.displayName = "Wrap";
}

var WrapItem = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded2$1);

  return /*#__PURE__*/React.createElement(chakra.li, _extends({
    ref: ref,
    __css: {
      display: "flex",
      alignItems: "flex-start"
    },
    className: cx("chakra-wrap__listitem", className)
  }, rest));
});

if (__DEV__) {
  WrapItem.displayName = "WrapItem";
}

var _excluded = ["isExternal", "target", "rel", "className"],
    _excluded2 = ["className"];
var LinkOverlay = /*#__PURE__*/forwardRef(function (props, ref) {
  var isExternal = props.isExternal,
      target = props.target,
      rel = props.rel,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  return /*#__PURE__*/React.createElement(chakra.a, _extends({}, rest, {
    ref: ref,
    className: cx("chakra-linkbox__overlay", className),
    rel: isExternal ? "noopener noreferrer" : rel,
    target: isExternal ? "_blank" : target,
    __css: {
      position: "static",
      "&::before": {
        content: "''",
        cursor: "inherit",
        display: "block",
        position: "absolute",
        top: 0,
        left: 0,
        zIndex: 0,
        width: "100%",
        height: "100%"
      }
    }
  }));
});

/**
 * `LinkBox` is used to wrap content areas within a link while ensuring semantic html
 *
 * @see Docs https://chakra-ui.com/docs/navigation/link-overlay
 * @see Resources https://www.sarasoueidan.com/blog/nested-links
 */
var LinkBox = /*#__PURE__*/forwardRef(function (props, ref) {
  var className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);

  return /*#__PURE__*/React.createElement(chakra.div, _extends({
    ref: ref,
    position: "relative"
  }, rest, {
    className: cx("chakra-linkbox", className),
    __css: {
      /* Elevate the links and abbreviations up */
      "a[href]:not(.chakra-linkbox__overlay), abbr[title]": {
        position: "relative",
        zIndex: 1
      }
    }
  }));
});

export { AbsoluteCenter, AspectRatio, Badge, Box, Center, Circle, Code, Container, Divider, Flex, Grid, GridItem, HStack, Heading, Kbd, Link, LinkBox, LinkOverlay, List, ListIcon, ListItem, OrderedList, SimpleGrid, Spacer, Square, Stack, StackDivider, StackItem, Text, UnorderedList, VStack, Wrap, WrapItem };
