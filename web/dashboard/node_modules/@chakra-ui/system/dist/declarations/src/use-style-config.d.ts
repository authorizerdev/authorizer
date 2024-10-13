import { SystemStyleObject } from "@chakra-ui/styled-system";
import { Dict } from "@chakra-ui/utils";
import { ThemingProps } from "./system.types";
export declare function useStyleConfig(themeKey: string, props: ThemingProps & Dict, opts: {
    isMultiPart: true;
}): Record<string, SystemStyleObject>;
export declare function useStyleConfig(themeKey: string, props?: ThemingProps & Dict, opts?: {
    isMultiPart?: boolean;
}): SystemStyleObject;
export declare function useMultiStyleConfig(themeKey: string, props: any): Record<string, import("@chakra-ui/styled-system").CSSObject>;
//# sourceMappingURL=use-style-config.d.ts.map