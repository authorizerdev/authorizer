import { StringOrNumber } from "@chakra-ui/utils";
import * as React from "react";
import { PropGetter } from "@chakra-ui/react-utils";
declare type EventOrValue = React.ChangeEvent<HTMLInputElement> | StringOrNumber;
export interface UseRadioGroupProps {
    /**
     * The value of the radio to be `checked`
     * (in controlled mode)
     */
    value?: StringOrNumber;
    /**
     * The value of the radio to be `checked`
     * initially (in uncontrolled mode)
     */
    defaultValue?: StringOrNumber;
    /**
     * Function called once a radio is checked
     * @param nextValue the value of the checked radio
     */
    onChange?(nextValue: string): void;
    /**
     * If `true`, all wrapped radio inputs will be disabled
     */
    isDisabled?: boolean;
    /**
     * If `true` and `isDisabled` is true, all wrapped radio inputs will remain
     * focusable but not interactive.
     */
    isFocusable?: boolean;
    /**
     * The `name` attribute forwarded to each `radio` element
     */
    name?: string;
    /**
     * If `true`, input elements will receive
     * `checked` attribute instead of `isChecked`.
     *
     * This assumes, you're using native radio inputs
     */
    isNative?: boolean;
}
declare type RadioPropGetter = PropGetter<HTMLInputElement, {
    onChange?: (e: EventOrValue) => void;
    value?: StringOrNumber;
    /**
     * checked is defined if isNative=true
     */
    checked?: boolean;
    /**
     * isChecked is defined if isNative=false
     */
    isChecked?: boolean;
} & Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "size" | "value">>;
/**
 * React hook to manage a group of radio inputs
 */
export declare function useRadioGroup(props?: UseRadioGroupProps): {
    getRootProps: PropGetter<any, {}>;
    getRadioProps: RadioPropGetter;
    name: string;
    ref: React.MutableRefObject<any>;
    focus: () => void;
    setValue: React.Dispatch<React.SetStateAction<StringOrNumber>>;
    value: StringOrNumber;
    onChange: (eventOrValue: EventOrValue) => void;
    isDisabled: boolean | undefined;
    isFocusable: boolean | undefined;
    htmlProps: {};
};
export declare type UseRadioGroupReturn = ReturnType<typeof useRadioGroup>;
export {};
//# sourceMappingURL=use-radio-group.d.ts.map