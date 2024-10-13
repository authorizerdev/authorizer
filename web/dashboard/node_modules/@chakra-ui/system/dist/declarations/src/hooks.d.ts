import { SystemStyleObject } from "@chakra-ui/styled-system";
import { Dict, StringOrNumber } from "@chakra-ui/utils";
import { ThemingProps } from "./system.types";
export declare function useChakra<T extends Dict = Dict>(): {
    theme: T;
    colorMode: import("@chakra-ui/color-mode").ColorMode;
    toggleColorMode: () => void;
    setColorMode: (value: any) => void;
};
export declare function useToken<T extends StringOrNumber>(scale: string, token: T | T[], fallback?: T | T[]): any;
export declare function useProps<P extends ThemingProps>(themeKey: string, props: P, isMulti: true): {
    styles: Record<string, SystemStyleObject>;
    props: Omit<P, keyof ThemingProps>;
};
export declare function useProps<P extends ThemingProps>(themeKey: string, props?: P, isMulti?: boolean): {
    styles: SystemStyleObject;
    props: Omit<P, keyof ThemingProps>;
};
//# sourceMappingURL=hooks.d.ts.map