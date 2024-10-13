export interface BaseBreakpointConfig {
    sm: string;
    md: string;
    lg: string;
    xl: string;
    "2xl"?: string;
    [key: string]: string | undefined;
}
export declare type Breakpoints<T> = T & {
    base: "0em";
};
export declare const createBreakpoints: <T extends BaseBreakpointConfig>(config: T) => Breakpoints<T>;
//# sourceMappingURL=create-breakpoints.d.ts.map