export declare type ColorMode = "light" | "dark";
/**
 * Function to add/remove class from `body` based on color mode
 */
export declare function syncBodyClassName(isDark: boolean, document: Document): void;
export declare const queries: {
    light: string;
    dark: string;
};
export declare const lightQuery: string;
export declare const darkQuery: string;
export declare function getColorScheme(fallback?: ColorMode): "dark" | "light";
/**
 * Adds system os color mode listener, and run the callback
 * once preference changes
 */
export declare function addListener(fn: (cm: ColorMode, isListenerEvent: true) => unknown): () => void;
export declare const root: {
    get: () => "" | ColorMode;
    set: (mode: ColorMode) => void;
};
//# sourceMappingURL=color-mode.utils.d.ts.map