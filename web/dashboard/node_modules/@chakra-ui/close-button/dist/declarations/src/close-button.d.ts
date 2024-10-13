import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
export interface CloseButtonProps extends HTMLChakraProps<"button">, ThemingProps<"CloseButton"> {
    /**
     * If `true`, the close button will be disabled.
     */
    isDisabled?: boolean;
}
/**
 * A button with a close icon.
 *
 * It is used to handle the close functionality in feedback and overlay components
 * like Alerts, Toasts, Drawers and Modals.
 */
export declare const CloseButton: import("@chakra-ui/system").ComponentWithAs<"button", CloseButtonProps>;
//# sourceMappingURL=close-button.d.ts.map