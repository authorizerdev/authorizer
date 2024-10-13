import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import * as React from "react";
import { UseSliderProps, UseSliderReturn } from "./use-slider";
interface SliderContext extends Omit<UseSliderReturn, "getInputProps" | "getRootProps"> {
}
declare const SliderProvider: React.Provider<SliderContext>, useSliderContext: () => SliderContext;
export { SliderProvider, useSliderContext };
export interface SliderProps extends UseSliderProps, ThemingProps<"Slider">, Omit<HTMLChakraProps<"div">, keyof UseSliderProps> {
}
/**
 * The Slider is used to allow users to make selections from a range of values.
 * It provides context and functionality for all slider components
 *
 * @see Docs     https://chakra-ui.com/docs/form/slider
 * @see WAI-ARIA https://www.w3.org/TR/wai-aria-practices/#slider
 */
export declare const Slider: import("@chakra-ui/system").ComponentWithAs<"div", SliderProps>;
export interface SliderThumbProps extends HTMLChakraProps<"div"> {
}
/**
 * Slider component that acts as the handle used to select predefined
 * values by dragging its handle along the track
 */
export declare const SliderThumb: import("@chakra-ui/system").ComponentWithAs<"div", SliderThumbProps>;
export interface SliderTrackProps extends HTMLChakraProps<"div"> {
}
export declare const SliderTrack: import("@chakra-ui/system").ComponentWithAs<"div", SliderTrackProps>;
export interface SliderInnerTrackProps extends HTMLChakraProps<"div"> {
}
export declare const SliderFilledTrack: import("@chakra-ui/system").ComponentWithAs<"div", SliderInnerTrackProps>;
export interface SliderMarkProps extends HTMLChakraProps<"div"> {
    value: number;
}
/**
 * SliderMark is used to provide names for specific Slider
 * values by defining labels or markers along the track.
 *
 * @see Docs https://chakra-ui.com/slider
 */
export declare const SliderMark: import("@chakra-ui/system").ComponentWithAs<"div", SliderMarkProps>;
//# sourceMappingURL=slider.d.ts.map