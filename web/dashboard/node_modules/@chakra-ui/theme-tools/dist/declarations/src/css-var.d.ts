export declare function isDecimal(value: any): boolean;
export declare function addPrefix(value: string, prefix?: string): string;
export declare function toVarRef(name: string, fallback?: string): string;
export declare function toVar(value: string, prefix?: string): string;
export declare type CSSVar = {
    variable: string;
    reference: string;
};
export declare type CSSVarOptions = {
    fallback?: string | CSSVar;
    prefix?: string;
};
export declare function cssVar(name: string, options?: CSSVarOptions): {
    variable: string;
    reference: string;
};
//# sourceMappingURL=css-var.d.ts.map