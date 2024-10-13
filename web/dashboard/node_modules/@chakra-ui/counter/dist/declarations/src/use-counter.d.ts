/// <reference types="react" />
import { StringOrNumber } from "@chakra-ui/utils";
export interface UseCounterProps {
    /**
     * The callback fired when the value changes
     */
    onChange?(valueAsString: string, valueAsNumber: number): void;
    /**
     * The number of decimal points used to round the value
     */
    precision?: number;
    /**
     * The initial value of the counter. Should be less than `max` and greater than `min`
     */
    defaultValue?: StringOrNumber;
    /**
     * The value of the counter. Should be less than `max` and greater than `min`
     */
    value?: StringOrNumber;
    /**
     * The step used to increment or decrement the value
     * @default 1
     */
    step?: number;
    /**
     * The minimum value of the counter
     * @default -Infinity
     */
    min?: number;
    /**
     * The maximum value of the counter
     * @default Infinity
     */
    max?: number;
    /**
     * This controls the value update behavior in general.
     *
     * - If `true` and you use the stepper or up/down arrow keys,
     *  the value will not exceed the `max` or go lower than `min`
     *
     * - If `false`, the value will be allowed to go out of range.
     *
     * @default true
     */
    keepWithinRange?: boolean;
}
export declare function useCounter(props?: UseCounterProps): {
    isOutOfRange: boolean;
    isAtMax: boolean;
    isAtMin: boolean;
    precision: number;
    value: StringOrNumber;
    valueAsNumber: number;
    update: (next: StringOrNumber) => void;
    reset: () => void;
    increment: (step?: any) => void;
    decrement: (step?: any) => void;
    clamp: (value: number) => string;
    cast: (value: StringOrNumber) => void;
    setValue: import("react").Dispatch<import("react").SetStateAction<StringOrNumber>>;
};
export declare type UseCounterReturn = ReturnType<typeof useCounter>;
//# sourceMappingURL=use-counter.d.ts.map