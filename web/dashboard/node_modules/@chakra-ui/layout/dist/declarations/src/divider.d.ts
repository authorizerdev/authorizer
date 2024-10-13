import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
/**
 * Layout component used to visually separate content in a list or group.
 * It display a thin horizontal or vertical line, and renders a `hr` tag.
 *
 * @see Docs https://chakra-ui.com/divider
 */
export declare const Divider: import("@chakra-ui/system").ComponentWithAs<"hr", DividerProps>;
export interface DividerProps extends HTMLChakraProps<"div">, ThemingProps<"Divider"> {
    orientation?: "horizontal" | "vertical";
}
//# sourceMappingURL=divider.d.ts.map