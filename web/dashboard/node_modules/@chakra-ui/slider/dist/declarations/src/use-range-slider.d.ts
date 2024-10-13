/// <reference types="react" />
import { PropGetter } from "@chakra-ui/react-utils";
export interface UseRangeSliderProps {
    /**
     * The minimum allowed value of the slider. Cannot be greater than max.
     * @default 0
     */
    min?: number;
    /**
     * The maximum allowed value of the slider. Cannot be less than min.
     * @default 100
     */
    max?: number;
    /**
     * The step in which increments/decrements have to be made
     * @default 1
     */
    step?: number;
    /**
     * The value of the slider in controlled mode
     */
    value?: number[];
    /**
     * The initial value of the slider in uncontrolled mode
     */
    defaultValue?: number[];
    /**
     * orientation of the slider
     * @default "horizontal"
     */
    orientation?: "horizontal" | "vertical";
    /**
     * If `true`, the value will be incremented or decremented in reverse.
     */
    isReversed?: boolean;
    /**
     * Function called when the user starts selecting a new value (by dragging or clicking)
     */
    onChangeStart?(value: number[]): void;
    /**
     * Function called when the user is done selecting a new value (by dragging or clicking)
     */
    onChangeEnd?(value: number[]): void;
    /**
     * Function called whenever the slider value changes  (by dragging or clicking)
     */
    onChange?(value: number[]): void;
    /**
     * The base `id` to use for the slider and its components
     */
    id?: string;
    /**
     * The name attribute of the hidden `input` field.
     * This is particularly useful in forms
     */
    name?: string | string[];
    /**
     * If `true`, the slider will be disabled
     */
    isDisabled?: boolean;
    /**
     * If `true`, the slider will be in `read-only` state
     */
    isReadOnly?: boolean;
    /**
     * Function that returns the `aria-valuetext` for screen readers.
     * It is mostly used to generate a more human-readable
     * representation of the value for assistive technologies
     */
    getAriaValueText?(value: number): string;
    /**
     * If `false`, the slider handle will not capture focus when value changes.
     * @default true
     */
    focusThumbOnChange?: boolean;
    /**
     * The static string to use used for `aria-valuetext`
     */
    "aria-valuetext"?: string[];
    /**
     * The static string to use used for `aria-label`
     * if no visible label is used.
     */
    "aria-label"?: string[];
    /**
     * The static string `aria-labelledby` that points to the
     * ID of the element that serves as label for the slider
     */
    "aria-labelledby"?: string[];
    /**
     * The writing mode
     */
    direction?: "ltr" | "rtl";
    /**
     * The minimum distance between slider thumbs. Useful for preventing
     * the thumbs from being too close together.
     * @default 0
     */
    minStepsBetweenThumbs?: number;
}
/**
 * React hook that implements an accessible range slider.
 *
 * It is an alternative to `<input type="range" />`, and returns
 * prop getters for the component parts
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices-1.1/#slider
 */
export declare function useRangeSlider(props: UseRangeSliderProps): {
    state: {
        value: number[];
        isFocused: boolean;
        isDragging: boolean;
        getThumbPercent: (i: number) => number;
        getThumbMinValue: (i: number) => number;
        getThumbMaxValue: (i: number) => number;
    };
    actions: {
        setValueAtIndex: (index: number, val: number) => void;
        setActiveIndex: import("react").Dispatch<import("react").SetStateAction<number>>;
        stepUp: (index: number, step?: number) => void;
        stepDown: (index: number, step?: number) => void;
        reset: () => void;
    };
    getRootProps: PropGetter<any, {}>;
    getTrackProps: PropGetter<any, {}>;
    getInnerTrackProps: PropGetter<any, {}>;
    getThumbProps: (props: any, ref?: any) => any;
    getMarkerProps: PropGetter<any, {
        value?: any;
    }>;
    getInputProps: (props: any, ref?: any) => any;
    getOutputProps: (props?: any, ref?: any) => any;
};
export declare type UseRangeSliderReturn = ReturnType<typeof useRangeSlider>;
//# sourceMappingURL=use-range-slider.d.ts.map