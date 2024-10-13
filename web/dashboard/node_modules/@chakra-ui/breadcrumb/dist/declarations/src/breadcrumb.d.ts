import { SystemProps, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
export interface BreadcrumbSeparatorProps extends HTMLChakraProps<"div"> {
    /**
     * @type SystemProps["mx"]
     */
    spacing?: SystemProps["mx"];
}
/**
 * React component that separates each breadcrumb link
 */
export declare const BreadcrumbSeparator: import("@chakra-ui/system").ComponentWithAs<"span", BreadcrumbSeparatorProps>;
export interface BreadcrumbLinkProps extends HTMLChakraProps<"a"> {
    isCurrentPage?: boolean;
}
/**
 * Breadcrumb link.
 *
 * It renders a `span` when it matches the current link. Otherwise,
 * it renders an anchor tag.
 */
export declare const BreadcrumbLink: import("@chakra-ui/system").ComponentWithAs<"a", BreadcrumbLinkProps>;
interface BreadcrumbItemOptions extends BreadcrumbOptions {
    isCurrentPage?: boolean;
    isLastChild?: boolean;
}
export interface BreadcrumbItemProps extends BreadcrumbItemOptions, HTMLChakraProps<"li"> {
}
/**
 * BreadcrumbItem is used to group a breadcrumb link.
 * It renders a `li` element to denote it belongs to an order list of links.
 *
 * @see Docs https://chakra-ui.com/breadcrumb
 */
export declare const BreadcrumbItem: import("@chakra-ui/system").ComponentWithAs<"li", BreadcrumbItemProps>;
export interface BreadcrumbOptions {
    /**
     * The visual separator between each breadcrumb item
     * @type string | React.ReactElement
     */
    separator?: string | React.ReactElement;
    /**
     * The left and right margin applied to the separator
     * @type SystemProps["mx"]
     */
    spacing?: SystemProps["mx"];
}
export interface BreadcrumbProps extends HTMLChakraProps<"nav">, BreadcrumbOptions, ThemingProps<"Breadcrumb"> {
}
/**
 * Breadcrumb is used to render a breadcrumb navigation landmark.
 * It renders a `nav` element with `aria-label` set to `Breadcrumb`
 *
 * @see Docs https://chakra-ui.com/breadcrumb
 */
export declare const Breadcrumb: import("@chakra-ui/system").ComponentWithAs<"nav", BreadcrumbProps>;
export {};
//# sourceMappingURL=breadcrumb.d.ts.map