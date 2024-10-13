import { HTMLChakraProps, SystemProps } from "@chakra-ui/system";
import * as React from "react";
interface ButtonSpinnerProps extends HTMLChakraProps<"div"> {
    label?: string;
    /**
     * @type SystemProps["margin"]
     */
    spacing?: SystemProps["margin"];
    placement?: "start" | "end";
}
export declare const ButtonSpinner: React.FC<ButtonSpinnerProps>;
export {};
//# sourceMappingURL=button-spinner.d.ts.map