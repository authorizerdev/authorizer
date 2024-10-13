import { HTMLChakraProps, SystemProps } from "@chakra-ui/system";
export interface WrapProps extends HTMLChakraProps<"div"> {
    /**
     * The space between the each child (even if it wraps)
     * @type SystemProps["margin"]
     */
    spacing?: SystemProps["margin"];
    /**
     * The `justify-content` value (for cross-axis alignment)
     * @type SystemProps["justifyContent"]
     */
    justify?: SystemProps["justifyContent"];
    /**
     * The `align-items` value (for main axis alignment)
     * @type SystemProps["alignItems"]
     */
    align?: SystemProps["alignItems"];
    /**
     * The `flex-direction` value
     * @type SystemProps["flexDirection"]
     */
    direction?: SystemProps["flexDirection"];
    /**
     * If `true`, the children will be wrapped in a `WrapItem`
     */
    shouldWrapChildren?: boolean;
}
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
export declare const Wrap: import("@chakra-ui/system").ComponentWithAs<"div", WrapProps>;
export interface WrapItemProps extends HTMLChakraProps<"li"> {
}
export declare const WrapItem: import("@chakra-ui/system").ComponentWithAs<"li", WrapItemProps>;
//# sourceMappingURL=wrap.d.ts.map