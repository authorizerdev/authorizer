import { FormControlOptions } from "@chakra-ui/form-control";
import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
interface TextareaOptions {
    /**
     * The border color when the textarea is focused. Use color keys in `theme.colors`
     * @example
     * focusBorderColor = "blue.500"
     */
    focusBorderColor?: string;
    /**
     * The border color when the textarea is invalid. Use color keys in `theme.colors`
     * @example
     * errorBorderColor = "red.500"
     */
    errorBorderColor?: string;
    /**
     * If `true`, the textarea element will span the full width of its parent
     *
     * @deprecated
     * This component defaults to 100% width,
     * please use the props `maxWidth` or `width` to configure
     */
    isFullWidth?: boolean;
}
declare type Omitted = "disabled" | "required" | "readOnly";
export interface TextareaProps extends Omit<HTMLChakraProps<"textarea">, Omitted>, TextareaOptions, FormControlOptions, ThemingProps<"Textarea"> {
}
/**
 * Textarea is used to enter an amount of text that's longer than a single line
 * @see Docs https://chakra-ui.com/textarea
 */
export declare const Textarea: import("@chakra-ui/system").ComponentWithAs<"textarea", TextareaProps>;
export {};
//# sourceMappingURL=textarea.d.ts.map