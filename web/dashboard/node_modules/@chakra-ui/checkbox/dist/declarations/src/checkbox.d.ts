import { HTMLChakraProps, PropsOf, SystemProps, ThemingProps } from "@chakra-ui/system";
import { Omit } from "@chakra-ui/utils";
import * as React from "react";
import { UseCheckboxProps } from "./use-checkbox";
declare type CheckboxControlProps = Omit<HTMLChakraProps<"div">, keyof UseCheckboxProps>;
declare type BaseInputProps = Pick<PropsOf<"input">, "onBlur" | "checked" | "defaultChecked">;
export interface CheckboxProps extends CheckboxControlProps, BaseInputProps, ThemingProps<"Checkbox">, UseCheckboxProps {
    /**
     * The spacing between the checkbox and its label text
     * @default 0.5rem
     * @type SystemProps["marginLeft"]
     */
    spacing?: SystemProps["marginLeft"];
    /**
     * The color of the checkbox icon when checked or indeterminate
     */
    iconColor?: string;
    /**
     * The size of the checkbox icon when checked or indeterminate
     */
    iconSize?: string | number;
    /**
     * The checked icon to use
     *
     * @type React.ReactElement
     * @default CheckboxIcon
     */
    icon?: React.ReactElement;
}
/**
 * Checkbox
 *
 * React component used in forms when a user needs to select
 * multiple values from several options.
 *
 * @see Docs https://chakra-ui.com/checkbox
 */
export declare const Checkbox: import("@chakra-ui/system").ComponentWithAs<"input", CheckboxProps>;
export {};
//# sourceMappingURL=checkbox.d.ts.map