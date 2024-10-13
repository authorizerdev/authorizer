import { Dict } from "@chakra-ui/utils";
export declare const transformFunctions: {
    filter(value: any): any;
    backdropFilter(value: any): any;
    ring(value: string): {
        "--chakra-ring-offset-shadow": string;
        "--chakra-ring-shadow": string;
        "--chakra-ring-width": any;
        boxShadow: string;
    };
    bgClip(value: string): {
        color: string;
        backgroundClip: string;
    } | {
        backgroundClip: string;
        color?: undefined;
    };
    transform(value: any): any;
    px(value: number | string): string;
    fraction(value: any): any;
    float(value: any, theme: Dict): any;
    degree(value: any): any;
    gradient: import("./types").Transform;
    blur: (value: any) => string;
    opacity: (value: any) => string;
    brightness: (value: any) => string;
    contrast: (value: any) => string;
    dropShadow: (value: any) => string;
    grayscale: (value: any) => string;
    hueRotate: (value: any) => string;
    invert: (value: any) => string;
    saturate: (value: any) => string;
    sepia: (value: any) => string;
    bgImage(value: any): any;
    outline(value: any): {
        outline: string;
        outlineOffset: string;
    } | {
        outline: any;
        outlineOffset?: undefined;
    };
    flexDirection(value: any): {
        flexDirection: any;
    };
};
//# sourceMappingURL=transform-functions.d.ts.map