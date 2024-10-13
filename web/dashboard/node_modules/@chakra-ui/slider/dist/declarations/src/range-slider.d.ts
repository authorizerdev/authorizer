import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import * as React from "react";
import { UseRangeSliderProps, UseRangeSliderReturn } from "./use-range-slider";
interface RangeSliderContext extends Omit<UseRangeSliderReturn, "getRootProps"> {
    name?: string | string[];
}
declare const RangeSliderProvider: React.Provider<RangeSliderContext>, useRangeSliderContext: () => RangeSliderContext;
export { RangeSliderProvider, useRangeSliderContext };
export interface RangeSliderProps extends UseRangeSliderProps, ThemingProps<"Slider">, Omit<HTMLChakraProps<"div">, keyof UseRangeSliderProps> {
}
/**
 * The Slider is used to allow users to make selections from a range of values.
 * It provides context and functionality for all slider components
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices/#slider
 */
export declare const RangeSlider: import("@chakra-ui/system").ComponentWithAs<"div", RangeSliderProps>;
export interface RangeSliderThumbProps extends HTMLChakraProps<"div"> {
    index: number;
}
/**
 * Slider component that acts as the handle used to select predefined
 * values by dragging its handle along the track
 */
export declare const RangeSliderThumb: import("@chakra-ui/system").ComponentWithAs<"div", RangeSliderThumbProps>;
export interface RangeSliderTrackProps extends HTMLChakraProps<"div"> {
}
export declare const RangeSliderTrack: import("@chakra-ui/system").ComponentWithAs<"div", RangeSliderTrackProps>;
export interface RangeSliderInnerTrackProps extends HTMLChakraProps<"div"> {
}
export declare const RangeSliderFilledTrack: import("@chakra-ui/system").ComponentWithAs<"div", RangeSliderInnerTrackProps>;
export interface RangeSliderMarkProps extends HTMLChakraProps<"div"> {
    value: number;
}
/**
 * SliderMark is used to provide names for specific Slider
 * values by defining labels or markers along the track.
 *
 * @see Docs https://chakra-ui.com/slider
 */
export declare const RangeSliderMark: import("@chakra-ui/system").ComponentWithAs<"div", RangeSliderMarkProps>;
//# sourceMappingURL=range-slider.d.ts.map