import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
import { UsePinInputProps } from "./use-pin-input";
interface InputOptions {
    /**
     * The border color when the input is focused. Use color keys in `theme.colors`
     * @example
     * focusBorderColor = "blue.500"
     */
    focusBorderColor?: string;
    /**
     * The border color when the input is invalid. Use color keys in `theme.colors`
     * @example
     * errorBorderColor = "red.500"
     */
    errorBorderColor?: string;
}
export interface PinInputProps extends UsePinInputProps, ThemingProps<"PinInput">, InputOptions {
    /**
     * The children of the pin input component
     */
    children: React.ReactNode;
}
export declare const PinInput: React.FC<PinInputProps>;
export interface PinInputFieldProps extends HTMLChakraProps<"input"> {
}
export declare const PinInputField: import("@chakra-ui/system").ComponentWithAs<"input", PinInputFieldProps>;
export {};
//# sourceMappingURL=pin-input.d.ts.map