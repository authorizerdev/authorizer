import { FormControlOptions } from "@chakra-ui/form-control";
import { PropsOf, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
declare type Omitted = "disabled" | "required" | "readOnly" | "size";
export interface SelectFieldProps extends Omit<HTMLChakraProps<"select">, Omitted> {
    isDisabled?: boolean;
}
export declare const SelectField: import("@chakra-ui/system").ComponentWithAs<"select", SelectFieldProps>;
interface RootProps extends Omit<HTMLChakraProps<"div">, "color"> {
}
interface SelectOptions extends FormControlOptions {
    /**
     * The border color when the select is focused. Use color keys in `theme.colors`
     * @example
     * focusBorderColor = "blue.500"
     */
    focusBorderColor?: string;
    /**
     * The border color when the select is invalid. Use color keys in `theme.colors`
     * @example
     * errorBorderColor = "red.500"
     */
    errorBorderColor?: string;
    /**
     * If `true`, the select element will span the full width of its parent
     *
     * @deprecated
     * This component defaults to 100% width,
     * please use the props `maxWidth` or `width` to configure
     */
    isFullWidth?: boolean;
    /**
     * The placeholder for the select. We render an `<option/>` element that has
     * empty value.
     *
     * ```jsx
     * <option value="">{placeholder}</option>
     * ```
     */
    placeholder?: string;
    /**
     * The size (width and height) of the icon
     */
    iconSize?: string;
    /**
     * The color of the icon
     */
    iconColor?: string;
}
export interface SelectProps extends SelectFieldProps, ThemingProps<"Select">, SelectOptions {
    /**
     * Props to forward to the root `div` element
     */
    rootProps?: RootProps;
    /**
     * The icon element to use in the select
     * @type React.ReactElement
     */
    icon?: React.ReactElement<any>;
}
/**
 * React component used to select one item from a list of options.
 */
export declare const Select: import("@chakra-ui/system").ComponentWithAs<"select", SelectProps>;
export declare const DefaultIcon: React.FC<PropsOf<"svg">>;
export {};
//# sourceMappingURL=select.d.ts.map