import { Dict } from "@chakra-ui/utils";
import * as CSS from "csstype";
import { StyleObjectOrFn } from "./system.types";
import { Config } from "./utils/prop-config";
import { CssTheme } from "./utils/types";
interface GetCSSOptions {
    theme: CssTheme;
    configs?: Config;
    pseudos?: Record<string, CSS.Pseudos | (string & {})>;
}
export declare function getCss(options: GetCSSOptions): (stylesOrFn: Dict, nested?: boolean) => Dict<any>;
export declare const css: (styles: StyleObjectOrFn) => (theme: any) => Dict<any>;
export {};
//# sourceMappingURL=css.d.ts.map