import type { CSSProperties } from "react";
export declare function getIds(id: string | number): {
    root: string;
    getThumb: (i: number) => string;
    getInput: (i: number) => string;
    track: string;
    innerTrack: string;
    getMarker: (i: number) => string;
    output: string;
};
declare type Orientation = "vertical" | "horizontal";
export declare function orient(options: {
    orientation: Orientation;
    vertical: CSSProperties;
    horizontal: CSSProperties;
}): CSSProperties;
declare type Size = {
    height: number;
    width: number;
};
export declare function getStyles(options: {
    orientation: Orientation;
    thumbPercents: number[];
    thumbRects: Size[];
    isReversed?: boolean;
}): {
    trackStyle: CSSProperties;
    innerTrackStyle: CSSProperties;
    rootStyle: CSSProperties;
    getThumbStyle: (i: number) => CSSProperties;
};
export declare function getIsReversed(options: {
    isReversed?: boolean;
    direction: "ltr" | "rtl";
    orientation?: "horizontal" | "vertical";
}): boolean | undefined;
export {};
//# sourceMappingURL=slider-utils.d.ts.map