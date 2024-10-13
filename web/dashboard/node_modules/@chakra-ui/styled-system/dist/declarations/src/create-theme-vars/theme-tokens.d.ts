import { Dict } from "@chakra-ui/utils";
declare const tokens: readonly ["colors", "borders", "borderWidths", "borderStyles", "fonts", "fontSizes", "fontWeights", "letterSpacings", "lineHeights", "radii", "space", "shadows", "sizes", "zIndices", "transition", "blur"];
export declare type ThemeScale = typeof tokens[number] | "transition.duration" | "transition.property" | "transition.easing";
export declare function extractTokens(theme: Dict): {
    [x: string]: any;
};
export declare function omitVars(rawTheme: Dict): {
    [x: string]: any;
};
export {};
//# sourceMappingURL=theme-tokens.d.ts.map