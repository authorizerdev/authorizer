'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var system = require('@chakra-ui/system');
var utils = require('@chakra-ui/utils');
var reactUtils = require('@chakra-ui/react-utils');
var React = require('react');

function _interopNamespace(e) {
  if (e && e.__esModule) return e;
  var n = Object.create(null);
  if (e) {
    Object.keys(e).forEach(function (k) {
      if (k !== 'default') {
        var d = Object.getOwnPropertyDescriptor(e, k);
        Object.defineProperty(n, k, d.get ? d : {
          enumerable: true,
          get: function () { return e[k]; }
        });
      }
    });
  }
  n["default"] = e;
  return Object.freeze(n);
}

var React__namespace = /*#__PURE__*/_interopNamespace(React);

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

var _excluded = ["spacing"],
    _excluded2 = ["isCurrentPage", "as", "className", "href"],
    _excluded3 = ["isCurrentPage", "separator", "isLastChild", "spacing", "children", "className"],
    _excluded4 = ["children", "spacing", "separator", "className"];

/**
 * React component that separates each breadcrumb link
 */
var BreadcrumbSeparator = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var spacing = props.spacing,
      rest = _objectWithoutPropertiesLoose(props, _excluded);

  var styles = system.useStyles();

  var separatorStyles = _extends({
    mx: spacing
  }, styles.separator);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
    ref: ref,
    role: "presentation"
  }, rest, {
    __css: separatorStyles
  }));
});

if (utils.__DEV__) {
  BreadcrumbSeparator.displayName = "BreadcrumbSeparator";
}

/**
 * Breadcrumb link.
 *
 * It renders a `span` when it matches the current link. Otherwise,
 * it renders an anchor tag.
 */
var BreadcrumbLink = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var isCurrentPage = props.isCurrentPage,
      as = props.as,
      className = props.className,
      href = props.href,
      rest = _objectWithoutPropertiesLoose(props, _excluded2);

  var styles = system.useStyles();

  var sharedProps = _extends({
    ref: ref,
    as: as,
    className: utils.cx("chakra-breadcrumb__link", className)
  }, rest);

  if (isCurrentPage) {
    return /*#__PURE__*/React__namespace.createElement(system.chakra.span, _extends({
      "aria-current": "page",
      __css: styles.link
    }, sharedProps));
  }

  return /*#__PURE__*/React__namespace.createElement(system.chakra.a, _extends({
    __css: styles.link,
    href: href
  }, sharedProps));
});

if (utils.__DEV__) {
  BreadcrumbLink.displayName = "BreadcrumbLink";
}

/**
 * BreadcrumbItem is used to group a breadcrumb link.
 * It renders a `li` element to denote it belongs to an order list of links.
 *
 * @see Docs https://chakra-ui.com/breadcrumb
 */
var BreadcrumbItem = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var isCurrentPage = props.isCurrentPage,
      separator = props.separator,
      isLastChild = props.isLastChild,
      spacing = props.spacing,
      children = props.children,
      className = props.className,
      rest = _objectWithoutPropertiesLoose(props, _excluded3);

  var validChildren = reactUtils.getValidChildren(children);
  var clones = validChildren.map(function (child) {
    if (child.type === BreadcrumbLink) {
      return /*#__PURE__*/React__namespace.cloneElement(child, {
        isCurrentPage: isCurrentPage
      });
    }

    if (child.type === BreadcrumbSeparator) {
      return /*#__PURE__*/React__namespace.cloneElement(child, {
        spacing: spacing,
        children: child.props.children || separator
      });
    }

    return child;
  });
  var styles = system.useStyles();

  var itemStyles = _extends({
    display: "inline-flex",
    alignItems: "center"
  }, styles.item);

  var _className = utils.cx("chakra-breadcrumb__list-item", className);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.li, _extends({
    ref: ref,
    className: _className
  }, rest, {
    __css: itemStyles
  }), clones, !isLastChild && /*#__PURE__*/React__namespace.createElement(BreadcrumbSeparator, {
    spacing: spacing
  }, separator));
});

if (utils.__DEV__) {
  BreadcrumbItem.displayName = "BreadcrumbItem";
}

/**
 * Breadcrumb is used to render a breadcrumb navigation landmark.
 * It renders a `nav` element with `aria-label` set to `Breadcrumb`
 *
 * @see Docs https://chakra-ui.com/breadcrumb
 */
var Breadcrumb = /*#__PURE__*/system.forwardRef(function (props, ref) {
  var styles = system.useMultiStyleConfig("Breadcrumb", props);
  var ownProps = system.omitThemingProps(props);

  var children = ownProps.children,
      _ownProps$spacing = ownProps.spacing,
      spacing = _ownProps$spacing === void 0 ? "0.5rem" : _ownProps$spacing,
      _ownProps$separator = ownProps.separator,
      separator = _ownProps$separator === void 0 ? "/" : _ownProps$separator,
      className = ownProps.className,
      rest = _objectWithoutPropertiesLoose(ownProps, _excluded4);

  var validChildren = reactUtils.getValidChildren(children);
  var count = validChildren.length;
  var clones = validChildren.map(function (child, index) {
    return /*#__PURE__*/React__namespace.cloneElement(child, {
      separator: separator,
      spacing: spacing,
      isLastChild: count === index + 1
    });
  });

  var _className = utils.cx("chakra-breadcrumb", className);

  return /*#__PURE__*/React__namespace.createElement(system.chakra.nav, _extends({
    ref: ref,
    "aria-label": "breadcrumb",
    className: _className,
    __css: styles.container
  }, rest), /*#__PURE__*/React__namespace.createElement(system.StylesProvider, {
    value: styles
  }, /*#__PURE__*/React__namespace.createElement(system.chakra.ol, {
    className: "chakra-breadcrumb__list"
  }, clones)));
});

if (utils.__DEV__) {
  Breadcrumb.displayName = "Breadcrumb";
}

exports.Breadcrumb = Breadcrumb;
exports.BreadcrumbItem = BreadcrumbItem;
exports.BreadcrumbLink = BreadcrumbLink;
exports.BreadcrumbSeparator = BreadcrumbSeparator;
