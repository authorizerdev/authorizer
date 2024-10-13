import * as React from "react";
import { ButtonProps } from "./button";
declare type OmittedProps = "leftIcon" | "isFullWidth" | "rightIcon" | "loadingText" | "iconSpacing" | "spinnerPlacement";
interface BaseButtonProps extends Omit<ButtonProps, OmittedProps> {
}
export interface IconButtonProps extends BaseButtonProps {
    /**
     * The icon to be used in the button.
     * @type React.ReactElement
     */
    icon?: React.ReactElement;
    /**
     * If `true`, the button will be perfectly round. Else, it'll be slightly round
     */
    isRound?: boolean;
    /**
     * A11y: A label that describes the button
     */
    "aria-label": string;
}
export declare const IconButton: import("@chakra-ui/system").ComponentWithAs<"button", IconButtonProps>;
export {};
//# sourceMappingURL=icon-button.d.ts.map