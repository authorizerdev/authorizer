import { WithCSSVar } from "@chakra-ui/styled-system";
import { Dict } from "@chakra-ui/utils";
import { ThemeProviderProps as EmotionThemeProviderProps } from "@emotion/react";
import * as React from "react";
export interface ThemeProviderProps extends EmotionThemeProviderProps {
    /**
     * The element to attach the CSS custom properties to.
     * @default ":host, :root"
     */
    cssVarsRoot?: string;
}
export declare const ThemeProvider: (props: ThemeProviderProps) => JSX.Element;
export declare function useTheme<T extends object = Dict>(): WithCSSVar<T>;
declare const StylesProvider: React.Provider<Dict<import("@chakra-ui/styled-system").CSSObject>>, useStyles: () => Dict<import("@chakra-ui/styled-system").CSSObject>;
export { StylesProvider, useStyles };
/**
 * Applies styles defined in `theme.styles.global` globally
 * using emotion's `Global` component
 */
export declare const GlobalStyle: () => JSX.Element;
//# sourceMappingURL=providers.d.ts.map