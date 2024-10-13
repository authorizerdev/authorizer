import { PropGetter } from "@chakra-ui/react-utils";
import React from "react";
export interface UseCheckboxProps {
    /**
     * If `true`, the checkbox will be checked.
     * You'll need to pass `onChange` to update its value (since it is now controlled)
     */
    isChecked?: boolean;
    /**
     * If `true`, the checkbox will be indeterminate.
     * This only affects the icon shown inside checkbox
     * and does not modify the isChecked property.
     */
    isIndeterminate?: boolean;
    /**
     * If `true`, the checkbox will be disabled
     */
    isDisabled?: boolean;
    /**
     * If `true` and `isDisabled` is passed, the checkbox will
     * remain tabbable but not interactive
     */
    isFocusable?: boolean;
    /**
     * If `true`, the checkbox will be readonly
     */
    isReadOnly?: boolean;
    /**
     * If `true`, the checkbox is marked as invalid.
     * Changes style of unchecked state.
     */
    isInvalid?: boolean;
    /**
     * If `true`, the checkbox input is marked as required,
     * and `required` attribute will be added
     */
    isRequired?: boolean;
    /**
     * If `true`, the checkbox will be initially checked.
     * @deprecated Please use the `defaultChecked` prop, which mirrors default
     * React checkbox behavior.
     */
    defaultIsChecked?: boolean;
    /**
     * If `true`, the checkbox will be initially checked.
     */
    defaultChecked?: boolean;
    /**
     * The callback invoked when the checked state of the `Checkbox` changes.
     */
    onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    /**
     * The callback invoked when the checkbox is blurred (loses focus)
     */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /**
     * The callback invoked when the checkbox is focused
     */
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /**
     * The name of the input field in a checkbox
     * (Useful for form submission).
     */
    name?: string;
    /**
     * The value to be used in the checkbox input.
     * This is the value that will be returned on form submission.
     */
    value?: string | number;
    /**
     * id assigned to input
     */
    id?: string;
    /**
     * Defines the string that labels the checkbox element.
     */
    "aria-label"?: string;
    /**
     * Refers to the `id` of the element that labels the checkbox element.
     */
    "aria-labelledby"?: string;
    "aria-invalid"?: true | undefined;
    "aria-describedby"?: string;
    tabIndex?: number;
}
/**
 * useCheckbox that provides all the state and focus management logic
 * for a checkbox. It is consumed by the `Checkbox` component
 *
 * @see Docs https://chakra-ui.com/checkbox#hooks
 */
export declare function useCheckbox(props?: UseCheckboxProps): {
    state: {
        isInvalid: boolean | undefined;
        isFocused: boolean;
        isChecked: boolean;
        isActive: boolean;
        isHovered: boolean;
        isIndeterminate: boolean | undefined;
        isDisabled: boolean | undefined;
        isReadOnly: boolean | undefined;
        isRequired: boolean | undefined;
    };
    getRootProps: PropGetter<any, {}>;
    getCheckboxProps: PropGetter<any, {}>;
    getInputProps: PropGetter<any, {}>;
    getLabelProps: PropGetter<any, {}>;
    htmlProps: {};
};
export declare type UseCheckboxReturn = ReturnType<typeof useCheckbox>;
//# sourceMappingURL=use-checkbox.d.ts.map