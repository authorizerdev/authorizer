import { HTMLChakraProps } from "@chakra-ui/system";
declare type Placement = "left" | "right";
export interface InputAddonProps extends HTMLChakraProps<"div"> {
    placement?: Placement;
}
/**
 * InputAddon
 *
 * Element to append or prepend to an input
 */
export declare const InputAddon: import("@chakra-ui/system").ComponentWithAs<"div", InputAddonProps>;
/**
 * InputLeftAddon
 *
 * Element to append to the left of an input
 */
export declare const InputLeftAddon: import("@chakra-ui/system").ComponentWithAs<"div", InputAddonProps>;
/**
 * InputRightAddon
 *
 * Element to append to the right of an input
 */
export declare const InputRightAddon: import("@chakra-ui/system").ComponentWithAs<"div", InputAddonProps>;
export {};
//# sourceMappingURL=input-addon.d.ts.map