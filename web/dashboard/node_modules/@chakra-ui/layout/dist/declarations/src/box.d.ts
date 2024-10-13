import { HTMLChakraProps } from "@chakra-ui/system";
export interface BoxProps extends HTMLChakraProps<"div"> {
}
/**
 * Box is the most abstract component on top of which other chakra
 * components are built. It renders a `div` element by default.
 *
 * @see Docs https://chakra-ui.com/box
 */
export declare const Box: import("@chakra-ui/system").ChakraComponent<"div", {}>;
/**
 * As a constraint, you can't pass size related props
 * Only `size` would be allowed
 */
declare type Omitted = "size" | "boxSize" | "width" | "height" | "w" | "h";
export interface SquareProps extends Omit<BoxProps, Omitted> {
    /**
     * The size (width and height) of the square
     */
    size?: BoxProps["width"];
    /**
     * If `true`, the content will be centered in the square
     */
    centerContent?: boolean;
}
export declare const Square: import("@chakra-ui/system").ComponentWithAs<"div", SquareProps>;
export declare const Circle: import("@chakra-ui/system").ComponentWithAs<"div", SquareProps>;
export {};
//# sourceMappingURL=box.d.ts.map