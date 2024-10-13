declare const typography: {
    letterSpacings: {
        tighter: string;
        tight: string;
        normal: string;
        wide: string;
        wider: string;
        widest: string;
    };
    lineHeights: {
        normal: string;
        none: number;
        shorter: number;
        short: number;
        base: number;
        tall: number;
        taller: string;
        "3": string;
        "4": string;
        "5": string;
        "6": string;
        "7": string;
        "8": string;
        "9": string;
        "10": string;
    };
    fontWeights: {
        hairline: number;
        thin: number;
        light: number;
        normal: number;
        medium: number;
        semibold: number;
        bold: number;
        extrabold: number;
        black: number;
    };
    fonts: {
        heading: string;
        body: string;
        mono: string;
    };
    fontSizes: {
        xs: string;
        sm: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
        "3xl": string;
        "4xl": string;
        "5xl": string;
        "6xl": string;
        "7xl": string;
        "8xl": string;
        "9xl": string;
    };
};
/**
 * @deprecated
 * You can derive the Typography type from the DefaultChakraTheme
 *
 * type Typography = Pick<
 *   DefaultChakraTheme,
 *   | "letterSpacings"
 *   | "lineHeights"
 *   | "fontWeights"
 *   | "fonts"
 *   | "fontSizes"
 *  >
 */
export declare type Typography = typeof typography;
export default typography;
//# sourceMappingURL=typography.d.ts.map