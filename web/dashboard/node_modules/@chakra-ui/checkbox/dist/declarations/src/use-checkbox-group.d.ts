import { Dict, StringOrNumber } from "@chakra-ui/utils";
import { ChangeEvent } from "react";
declare type EventOrValue = ChangeEvent<HTMLInputElement> | StringOrNumber;
export interface UseCheckboxGroupProps {
    /**
     * The value of the checkbox group
     */
    value?: StringOrNumber[];
    /**
     * The initial value of the checkbox group
     */
    defaultValue?: StringOrNumber[];
    /**
     * The callback fired when any children Checkbox is checked or unchecked
     */
    onChange?(value: StringOrNumber[]): void;
    /**
     * If `true`, all wrapped checkbox inputs will be disabled
     */
    isDisabled?: boolean;
    /**
     * If `true`, input elements will receive
     * `checked` attribute instead of `isChecked`.
     *
     * This assumes, you're using native radio inputs
     */
    isNative?: boolean;
}
/**
 * React hook that provides all the state management logic
 * for a group of checkboxes.
 *
 * It is consumed by the `CheckboxGroup` component
 */
export declare function useCheckboxGroup(props?: UseCheckboxGroupProps): {
    value: StringOrNumber[];
    isDisabled: boolean | undefined;
    onChange: (eventOrValue: EventOrValue) => void;
    setValue: import("react").Dispatch<import("react").SetStateAction<StringOrNumber[]>>;
    getCheckboxProps: (props?: Dict) => {
        [x: string]: any;
        onChange: (eventOrValue: EventOrValue) => void;
    };
};
export declare type UseCheckboxGroupReturn = ReturnType<typeof useCheckboxGroup>;
export {};
//# sourceMappingURL=use-checkbox-group.d.ts.map