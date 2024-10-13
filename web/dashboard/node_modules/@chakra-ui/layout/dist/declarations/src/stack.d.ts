import { ChakraComponent, HTMLChakraProps, SystemProps } from "@chakra-ui/system";
import * as React from "react";
import type { StackDirection } from "./stack.utils";
export type { StackDirection };
interface StackOptions {
    /**
     * Shorthand for `alignItems` style prop
     * @type SystemProps["alignItems"]
     */
    align?: SystemProps["alignItems"];
    /**
     * Shorthand for `justifyContent` style prop
     * @type SystemProps["justifyContent"]
     */
    justify?: SystemProps["justifyContent"];
    /**
     * Shorthand for `flexWrap` style prop
     * @type SystemProps["flexWrap"]
     */
    wrap?: SystemProps["flexWrap"];
    /**
     * The space between each stack item
     * @type SystemProps["margin"]
     */
    spacing?: SystemProps["margin"];
    /**
     * The direction to stack the items.
     */
    direction?: StackDirection;
    /**
     * If `true`, each stack item will show a divider
     * @type React.ReactElement
     */
    divider?: React.ReactElement;
    /**
     * If `true`, the children will be wrapped in a `Box` with
     * `display: inline-block`, and the `Box` will take the spacing props
     */
    shouldWrapChildren?: boolean;
    /**
     * If `true` the items will be stacked horizontally.
     */
    isInline?: boolean;
}
export interface StackDividerProps extends HTMLChakraProps<"div"> {
}
export declare const StackDivider: ChakraComponent<"div">;
export declare const StackItem: ChakraComponent<"div">;
export interface StackProps extends HTMLChakraProps<"div">, StackOptions {
}
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
export declare const Stack: import("@chakra-ui/system").ComponentWithAs<"div", StackProps>;
/**
 * A view that arranges its children in a horizontal line.
 */
export declare const HStack: import("@chakra-ui/system").ComponentWithAs<"div", StackProps>;
/**
 * A view that arranges its children in a vertical line.
 */
export declare const VStack: import("@chakra-ui/system").ComponentWithAs<"div", StackProps>;
//# sourceMappingURL=stack.d.ts.map