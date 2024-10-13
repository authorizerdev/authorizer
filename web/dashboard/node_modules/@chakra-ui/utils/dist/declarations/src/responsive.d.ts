import { Dict } from "./types";
export declare const breakpoints: readonly string[];
export declare function mapResponsive(prop: any, mapper: (val: any) => any): any;
export declare function objectToArrayNotation(obj: Dict, bps?: readonly string[]): any[];
export declare function arrayToObjectNotation(values: any[], bps?: readonly string[]): Dict<any>;
export declare function isResponsiveObjectLike(obj: Dict, bps?: readonly string[]): boolean;
/**
 * since breakpoints are defined as custom properties on an array, you may
 * `Object.keys(theme.breakpoints)` to retrieve both regular numeric indices
 * and custom breakpoints as string.
 *
 * This function returns true given a custom array property.
 */
export declare const isCustomBreakpoint: (maybeBreakpoint: string) => boolean;
//# sourceMappingURL=responsive.d.ts.map