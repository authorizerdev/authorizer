import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import { UseNumberInputProps } from "./use-number-input";
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
     * please use the props `maxWidth` or `width` to configure
     */
    isFullWidth?: boolean;
}
export interface NumberInputProps extends UseNumberInputProps, ThemingProps<"NumberInput">, InputOptions, Omit<HTMLChakraProps<"div">, keyof UseNumberInputProps> {
}
/**
 * NumberInput
 *
 * React component that provides context and logic to all
 * number input sub-components.
 *
 * It renders a `div` by default.
 *
 * @see Docs http://chakra-ui.com/numberinput
 */
export declare const NumberInput: import("@chakra-ui/system").ComponentWithAs<"div", NumberInputProps>;
export interface NumberInputStepperProps extends HTMLChakraProps<"div"> {
}
/**
 * NumberInputStepper
 *
 * React component used to group the increment and decrement
 * button spinners.
 *
 * It renders a `div` by default.
 *
 * @see Docs http://chakra-ui.com/components/number-input
 */
export declare const NumberInputStepper: import("@chakra-ui/system").ComponentWithAs<"div", NumberInputStepperProps>;
export interface NumberInputFieldProps extends HTMLChakraProps<"input"> {
}
/**
 * NumberInputField
 *
 * React component that represents the actual `input` field
 * where users can type to edit numeric values.
 *
 * It renders an `input` by default and ensures only numeric
 * values can be typed.
 *
 * @see Docs http://chakra-ui.com/numberinput
 */
export declare const NumberInputField: import("@chakra-ui/system").ComponentWithAs<"input", NumberInputFieldProps>;
export declare const StyledStepper: import("@chakra-ui/system").ChakraComponent<"div", {}>;
export interface NumberDecrementStepperProps extends HTMLChakraProps<"div"> {
}
/**
 * NumberDecrementStepper
 *
 * React component used to decrement the number input's value
 *
 * It renders a `div` with `role=button` by default
 */
export declare const NumberDecrementStepper: import("@chakra-ui/system").ComponentWithAs<"div", NumberDecrementStepperProps>;
export interface NumberIncrementStepperProps extends HTMLChakraProps<"div"> {
}
/**
 * NumberIncrementStepper
 *
 * React component used to increment the number input's value
 *
 * It renders a `div` with `role=button` by default
 */
export declare const NumberIncrementStepper: import("@chakra-ui/system").ComponentWithAs<"div", NumberIncrementStepperProps>;
export {};
//# sourceMappingURL=number-input.d.ts.map