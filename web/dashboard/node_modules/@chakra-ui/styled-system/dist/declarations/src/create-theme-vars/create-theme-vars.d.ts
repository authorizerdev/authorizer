import { Dict } from "@chakra-ui/utils";
export interface CreateThemeVarsOptions {
    cssVarPrefix?: string;
}
export interface ThemeVars {
    cssVars: Dict;
    cssMap: Dict;
}
export declare function createThemeVars(target: Dict, options: CreateThemeVarsOptions): ThemeVars;
//# sourceMappingURL=create-theme-vars.d.ts.map