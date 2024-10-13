import { FormControlOptions } from "@chakra-ui/form-control";
import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
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
    /**
     * If `true`, the input element will span the full width of its parent
     *
     * @deprecated
     * This component defaults to 100% width,
     *  please use the props `maxWidth` or `width` to configure
     */
    isFullWidth?: boolean;
}
declare type Omitted = "disabled" | "required" | "readOnly" | "size";
export interface InputProps extends Omit<HTMLChakraProps<"input">, Omitted>, InputOptions, ThemingProps<"Input">, FormControlOptions {
}
/**
 * Input
 *
 * Element that allows users enter single valued data.
 */
export declare const Input: import("@chakra-ui/system").ComponentWithAs<"input", InputProps>;
export {};
//# sourceMappingURL=input.d.ts.map