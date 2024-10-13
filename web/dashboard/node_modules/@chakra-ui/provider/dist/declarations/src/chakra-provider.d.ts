import { ColorModeProviderProps, ThemeProviderProps } from "@chakra-ui/system";
import { Dict } from "@chakra-ui/utils";
import { EnvironmentProviderProps } from "@chakra-ui/react-env";
import * as React from "react";
export interface ChakraProviderProps extends Pick<ThemeProviderProps, "cssVarsRoot"> {
    /**
     * a theme. if omitted, uses the default theme provided by chakra
     */
    theme?: Dict;
    /**
     * Common z-index to use for `Portal`
     *
     * @default undefined
     */
    portalZIndex?: number;
    /**
     * If `true`, `CSSReset` component will be mounted to help
     * you reset browser styles
     *
     * @default true
     */
    resetCSS?: boolean;
    /**
     * manager to persist a users color mode preference in
     *
     * omit if you don't render server-side
     * for SSR: choose `cookieStorageManager`
     *
     * @default localStorageManager
     */
    colorModeManager?: ColorModeProviderProps["colorModeManager"];
    /**
     * Your application content
     */
    children?: React.ReactNode;
    /**
     * The environment (`window` and `document`) to be used by
     * all components and hooks.
     *
     * By default, we smartly determine the ownerDocument and defaultView
     * based on where `ChakraProvider` is rendered.
     */
    environment?: EnvironmentProviderProps["environment"];
}
/**
 * The global provider that must be added to make all Chakra components
 * work correctly
 */
export declare const ChakraProvider: React.FC<ChakraProviderProps>;
//# sourceMappingURL=chakra-provider.d.ts.map