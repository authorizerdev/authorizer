import type { ThemeScale } from "../create-theme-vars";
import { logical, PropConfig } from "./prop-config";
import { transformFunctions as transforms } from "./transform-functions";
export { transforms };
export * from "./types";
export declare const t: {
    borderWidths: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    borderStyles: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    colors: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    borders: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    radii: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    space: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    spaceT: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    degreeT(property: PropConfig["property"]): {
        property: ((((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>) | ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>)[]) | ((theme: import("./types").CssTheme) => ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>) | ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>)[])) | undefined;
        transform: (value: any) => any;
    };
    prop(property: PropConfig["property"], scale?: ThemeScale | undefined, transform?: PropConfig["transform"]): {
        transform?: import("./types").Transform | undefined;
        property: ((((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>) | ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>)[]) | ((theme: import("./types").CssTheme) => ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>) | ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>)[])) | undefined;
        scale: ThemeScale | undefined;
    };
    propT(property: PropConfig["property"], transform?: PropConfig["transform"]): {
        property: ((((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>) | ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>)[]) | ((theme: import("./types").CssTheme) => ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>) | ((string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>)[])) | undefined;
        transform: import("./types").Transform | undefined;
    };
    sizes: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    sizesT: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    shadows: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
    logical: typeof logical;
    blur: <T extends (string & {}) | keyof import("csstype").Properties<0 | (string & {}), string & {}>>(property: T | T[]) => PropConfig;
};
//# sourceMappingURL=index.d.ts.map