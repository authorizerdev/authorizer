import * as React from "react";
import { ColorMode } from "./color-mode.utils";
import { StorageManager } from "./storage-manager";
declare type ConfigColorMode = ColorMode | "system" | undefined;
export type { ColorMode, ConfigColorMode };
export interface ColorModeOptions {
    initialColorMode?: ConfigColorMode;
    useSystemColorMode?: boolean;
}
interface ColorModeContextType {
    colorMode: ColorMode;
    toggleColorMode: () => void;
    setColorMode: (value: any) => void;
}
export declare const ColorModeContext: React.Context<ColorModeContextType>;
/**
 * React hook that reads from `ColorModeProvider` context
 * Returns the color mode and function to toggle it
 */
export declare const useColorMode: () => ColorModeContextType;
export interface ColorModeProviderProps {
    value?: ColorMode;
    children?: React.ReactNode;
    options: ColorModeOptions;
    colorModeManager?: StorageManager;
}
/**
 * Provides context for the color mode based on config in `theme`
 * Returns the color mode and function to toggle the color mode
 */
export declare function ColorModeProvider(props: ColorModeProviderProps): JSX.Element;
export declare namespace ColorModeProvider {
    var displayName: string;
}
/**
 * Locks the color mode to `dark`, without any way to change it.
 */
export declare const DarkMode: React.FC;
/**
 * Locks the color mode to `light` without any way to change it.
 */
export declare const LightMode: React.FC;
/**
 * Change value based on color mode.
 *
 * @param light the light mode value
 * @param dark the dark mode value
 *
 * @example
 *
 * ```js
 * const Icon = useColorModeValue(MoonIcon, SunIcon)
 * ```
 */
export declare function useColorModeValue<TLight = unknown, TDark = unknown>(light: TLight, dark: TDark): TLight | TDark;
//# sourceMappingURL=color-mode-provider.d.ts.map