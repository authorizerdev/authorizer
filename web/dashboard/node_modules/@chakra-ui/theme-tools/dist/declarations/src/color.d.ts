import { WCAG2Parms } from "@ctrl/tinycolor";
import { Dict } from "@chakra-ui/utils";
/**
 * Get the color raw value from theme
 * @param theme - the theme object
 * @param color - the color path ("green.200")
 * @param fallback - the fallback color
 */
export declare const getColor: (theme: Dict, color: string, fallback?: string | undefined) => any;
/**
 * Determines if the tone of given color is "light" or "dark"
 * @param color - the color in hex, rgb, or hsl
 */
export declare const tone: (color: string) => (theme: Dict) => "dark" | "light";
/**
 * Determines if a color tone is "dark"
 * @param color - the color in hex, rgb, or hsl
 */
export declare const isDark: (color: string) => (theme: Dict) => boolean;
/**
 * Determines if a color tone is "light"
 * @param color - the color in hex, rgb, or hsl
 */
export declare const isLight: (color: string) => (theme: Dict) => boolean;
/**
 * Make a color transparent
 * @param color - the color in hex, rgb, or hsl
 * @param opacity - the amount of opacity the color should have (0-1)
 */
export declare const transparentize: (color: string, opacity: number) => (theme: Dict) => string;
/**
 * Add white to a color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount white to add (0-100)
 */
export declare const whiten: (color: string, amount: number) => (theme: Dict) => string;
/**
 * Add black to a color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount black to add (0-100)
 */
export declare const blacken: (color: string, amount: number) => (theme: Dict) => string;
/**
 * Darken a specified color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount to darken (0-100)
 */
export declare const darken: (color: string, amount: number) => (theme: Dict) => string;
/**
 * Lighten a specified color
 * @param color - the color in hex, rgb, or hsl
 * @param amount - the amount to lighten (0-100)
 */
export declare const lighten: (color: string, amount: number) => (theme: Dict) => string;
/**
 * Checks the contract ratio of between 2 colors,
 * based on the Web Content Accessibility Guidelines (Version 2.0).
 *
 * @param fg - the foreground or text color
 * @param bg - the background color
 */
export declare const contrast: (fg: string, bg: string) => (theme: Dict) => number;
/**
 * Checks if a color meets the Web Content Accessibility
 * Guidelines (Version 2.0) for constract ratio.
 *
 * @param fg - the foreground or text color
 * @param bg - the background color
 */
export declare const isAccessible: (textColor: string, bgColor: string, options?: WCAG2Parms | undefined) => (theme: Dict) => boolean;
export declare const complementary: (color: string) => (theme: Dict) => string;
export declare function generateStripe(size?: string, color?: string): {
    backgroundImage: string;
    backgroundSize: string;
};
interface RandomColorOptions {
    /**
     * If passed, string will be used to generate
     * random color
     */
    string?: string;
    /**
     * List of colors to pick from at random
     */
    colors?: string[];
}
export declare function randomColor(opts?: RandomColorOptions): string;
export {};
//# sourceMappingURL=color.d.ts.map