import { ThemingProps } from "@chakra-ui/system";
import * as React from "react";
import { UseCheckboxGroupProps, UseCheckboxGroupReturn } from "./use-checkbox-group";
export interface CheckboxGroupProps extends UseCheckboxGroupProps, Omit<ThemingProps<"Checkbox">, "orientation"> {
    children?: React.ReactNode;
}
export interface CheckboxGroupContext extends Pick<UseCheckboxGroupReturn, "onChange" | "value" | "isDisabled">, Omit<ThemingProps<"Checkbox">, "orientation"> {
}
declare const useCheckboxGroupContext: () => CheckboxGroupContext;
export { useCheckboxGroupContext };
/**
 * Used for multiple checkboxes which are bound in one group,
 * and it indicates whether one or more options are selected.
 *
 * @see Docs https://chakra-ui.com/checkbox
 */
export declare const CheckboxGroup: React.FC<CheckboxGroupProps>;
//# sourceMappingURL=checkbox-group.d.ts.map