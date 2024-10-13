/// <reference types="react" />
import { HTMLChakraProps } from "@chakra-ui/system";
import { SlideOptions } from "@chakra-ui/transition";
import { ModalProps } from "./modal";
declare type LogicalPlacement = "start" | "end";
declare type DrawerPlacement = SlideOptions["direction"] | LogicalPlacement;
interface DrawerOptions {
    /**
     * The placement of the drawer
     */
    placement?: DrawerPlacement;
    /**
     * If `true` and drawer's placement is `top` or `bottom`,
     * the drawer will occupy the viewport height (100vh)
     */
    isFullHeight?: boolean;
}
export interface DrawerProps extends DrawerOptions, Omit<ModalProps, "scrollBehavior" | "motionPreset" | "isCentered"> {
}
export declare function Drawer(props: DrawerProps): JSX.Element;
export interface DrawerContentProps extends HTMLChakraProps<"section"> {
}
/**
 * ModalContent is used to group modal's content. It has all the
 * necessary `aria-*` properties to indicate that it is a modal
 */
export declare const DrawerContent: import("@chakra-ui/system").ComponentWithAs<"section", DrawerContentProps>;
export { ModalBody as DrawerBody, ModalCloseButton as DrawerCloseButton, ModalFooter as DrawerFooter, ModalHeader as DrawerHeader, ModalOverlay as DrawerOverlay, } from "./modal";
//# sourceMappingURL=drawer.d.ts.map