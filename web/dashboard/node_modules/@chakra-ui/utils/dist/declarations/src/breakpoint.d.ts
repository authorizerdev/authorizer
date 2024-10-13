import { Dict } from "./types";
export declare function px(value: number | string | null): string | null;
export declare function analyzeBreakpoints(breakpoints: Dict): {
    keys: Set<string>;
    normalized: string[];
    isResponsive(test: Dict): boolean;
    asObject: Dict<any>;
    asArray: string[];
    details: {
        breakpoint: string;
        minW: any;
        maxW: any;
        maxWQuery: string;
        minWQuery: string;
        minMaxQuery: string;
    }[];
    media: (string | null)[];
    toArrayValue(test: Dict): any[];
    toObjectValue(test: any[]): any;
} | null;
export declare type AnalyzeBreakpointsReturn = ReturnType<typeof analyzeBreakpoints>;
//# sourceMappingURL=breakpoint.d.ts.map