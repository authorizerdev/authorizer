import { HTMLChakraProps } from "@chakra-ui/system";
export interface CenterProps extends HTMLChakraProps<"div"> {
}
/**
 * React component used to horizontally and vertically center its child.
 * It uses the popular `display: flex` centering technique.
 *
 * @see Docs https://chakra-ui.com/center
 */
export declare const Center: import("@chakra-ui/system").ChakraComponent<"div", {}>;
export interface AbsoluteCenterProps extends HTMLChakraProps<"div"> {
    axis?: "horizontal" | "vertical" | "both";
}
/**
 * React component used to horizontally and vertically center an element
 * relative to its parent dimensions.
 *
 * It uses the `position: absolute` strategy.
 *
 * @see Docs https://chakra-ui.com/center
 * @see WebDev https://web.dev/centering-in-css/#5.-pop-and-plop
 */
export declare const AbsoluteCenter: import("@chakra-ui/system").ComponentWithAs<"div", AbsoluteCenterProps>;
//# sourceMappingURL=center.d.ts.map