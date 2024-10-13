import { IconProps } from "@chakra-ui/icon";
import { SystemProps, HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
interface ListOptions {
    /**
     * Short hand prop for `listStyleType`
     * @type SystemProps["listStyleType"]
     */
    styleType?: SystemProps["listStyleType"];
    /**
     * Short hand prop for `listStylePosition`
     * @type SystemProps["listStylePosition"]
     */
    stylePosition?: SystemProps["listStylePosition"];
    /**
     * The space between each list item
     * @type SystemProps["margin"]
     */
    spacing?: SystemProps["margin"];
}
export interface ListProps extends HTMLChakraProps<"ul">, ThemingProps<"List">, ListOptions {
}
/**
 * List is used to display list items, it renders a `<ul>` by default.
 *
 * @see Docs https://chakra-ui.com/list
 */
export declare const List: import("@chakra-ui/system").ComponentWithAs<"ul", ListProps>;
export declare const OrderedList: import("@chakra-ui/system").ComponentWithAs<"ol", ListProps>;
export declare const UnorderedList: import("@chakra-ui/system").ComponentWithAs<"ul", ListProps>;
export interface ListItemProps extends HTMLChakraProps<"li"> {
}
/**
 * ListItem
 *
 * Used to render a list item
 */
export declare const ListItem: import("@chakra-ui/system").ComponentWithAs<"li", ListItemProps>;
/**
 * ListIcon
 *
 * Used to render an icon beside the list item text
 */
export declare const ListIcon: import("@chakra-ui/system").ComponentWithAs<"svg", IconProps>;
export {};
//# sourceMappingURL=list.d.ts.map