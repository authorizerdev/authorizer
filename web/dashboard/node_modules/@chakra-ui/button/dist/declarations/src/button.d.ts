import { HTMLChakraProps, SystemProps, ThemingProps } from "@chakra-ui/system";
import * as React from "react";
export interface ButtonOptions {
    /**
     * If `true`, the button will show a spinner.
     */
    isLoading?: boolean;
    /**
     * If `true`, the button will be styled in its active state.
     */
    isActive?: boolean;
    /**
     * If `true`, the button will be disabled.
     */
    isDisabled?: boolean;
    /**
     * The label to show in the button when `isLoading` is true
     * If no text is passed, it only shows the spinner
     */
    loadingText?: string;
    /**
     * If `true`, the button will take up the full width of its container.
     */
    isFullWidth?: boolean;
    /**
     * The html button type to use.
     */
    type?: "button" | "reset" | "submit";
    /**
     * If added, the button will show an icon before the button's label.
     * @type React.ReactElement
     */
    leftIcon?: React.ReactElement;
    /**
     * If added, the button will show an icon after the button's label.
     * @type React.ReactElement
     */
    rightIcon?: React.ReactElement;
    /**
     * The space between the button icon and label.
     * @type SystemProps["marginRight"]
     */
    iconSpacing?: SystemProps["marginRight"];
    /**
     * Replace the spinner component when `isLoading` is set to `true`
     * @type React.ReactElement
     */
    spinner?: React.ReactElement;
    /**
     * It determines the placement of the spinner when isLoading is true
     * @default "start"
     */
    spinnerPlacement?: "start" | "end";
}
export interface ButtonProps extends HTMLChakraProps<"button">, ButtonOptions, ThemingProps<"Button"> {
}
export declare const Button: import("@chakra-ui/system").ComponentWithAs<"button", ButtonProps>;
//# sourceMappingURL=button.d.ts.map