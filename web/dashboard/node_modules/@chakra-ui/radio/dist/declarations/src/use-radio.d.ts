import { PropGetter } from "@chakra-ui/react-utils";
import { ChangeEvent } from "react";
/**
 * @todo use the `useClickable` hook here
 * to manage the isFocusable & isDisabled props
 */
export interface UseRadioProps {
    /**
     * id assigned to input
     */
    id?: string;
    /**
     * The name of the input field in a radio
     * (Useful for form submission).
     */
    name?: string;
    /**
     * The value to be used in the radio button.
     * This is the value that will be returned on form submission.
     */
    value?: string | number;
    /**
     * If `true`, the radio will be checked.
     * You'll need to pass `onChange` to update its value (since it is now controlled)
     */
    isChecked?: boolean;
    /**
     * If `true`, the radio will be initially checked.
     *
     * @deprecated Please use `defaultChecked` which mirrors the default prop
     * name for radio elements.
     */
    defaultIsChecked?: boolean;
    /**
     * If `true`, the radio will be initially checked.
     */
    defaultChecked?: boolean;
    /**
     * If `true`, the radio will be disabled
     */
    isDisabled?: boolean;
    /**
     * If `true` and `isDisabled` is true, the radio will remain
     * focusable but not interactive.
     */
    isFocusable?: boolean;
    /**
     * If `true`, the radio will be read-only
     */
    isReadOnly?: boolean;
    /**
     * If `true`, the radio button will be invalid. This also sets `aria-invalid` to `true`.
     */
    isInvalid?: boolean;
    /**
     * If `true`, the radio button will be required. This also sets `aria-required` to `true`.
     */
    isRequired?: boolean;
    /**
     * Function called when checked state of the `input` changes
     */
    onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
    /**
     * @internal
     */
    "data-radiogroup"?: any;
}
export declare function useRadio(props?: UseRadioProps): {
    state: {
        isInvalid: boolean;
        isFocused: boolean;
        isChecked: boolean;
        isActive: boolean;
        isHovered: boolean;
        isDisabled: boolean;
        isReadOnly: boolean;
        isRequired: boolean;
    };
    getCheckboxProps: PropGetter<any, {}>;
    getInputProps: PropGetter<HTMLInputElement, {}>;
    getLabelProps: PropGetter<any, {}>;
    getRootProps: PropGetter<any, {}>;
    htmlProps: {};
};
export declare type UseRadioReturn = ReturnType<typeof useRadio>;
//# sourceMappingURL=use-radio.d.ts.map