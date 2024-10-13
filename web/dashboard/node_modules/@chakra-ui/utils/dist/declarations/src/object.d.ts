import type { Dict, Omit } from "./types";
export { default as mergeWith } from "lodash.mergewith";
export declare function omit<T extends Dict, K extends keyof T>(object: T, keys: K[]): Omit<T, K>;
export declare function pick<T extends Dict, K extends keyof T>(object: T, keys: K[]): { [P in K]: T[P]; };
export declare function split<T extends Dict, K extends keyof T>(object: T, keys: K[]): [{ [P in K]: T[P]; }, Omit<T, K>];
/**
 * Get value from a deeply nested object using a string path.
 * Memoizes the value.
 * @param obj - the object
 * @param path - the string path
 * @param def  - the fallback value
 */
export declare function get(obj: object, path: string | number, fallback?: any, index?: number): any;
declare type Get = (obj: Readonly<object>, path: string | number, fallback?: any, index?: number) => any;
export declare const memoize: (fn: Get) => Get;
export declare const memoizedGet: Get;
/**
 * Get value from deeply nested object, based on path
 * It returns the path value if not found in object
 *
 * @param path - the string path or value
 * @param scale - the string path or value
 */
export declare function getWithDefault(path: any, scale: any): any;
declare type FilterFn<T> = (value: any, key: string, object: T) => boolean;
/**
 * Returns the items of an object that meet the condition specified in a callback function.
 *
 * @param object the object to loop through
 * @param fn The filter function
 */
export declare function objectFilter<T extends Dict>(object: T, fn: FilterFn<T>): Dict<any>;
export declare const filterUndefined: (object: Dict) => Dict<any>;
export declare const objectKeys: <T extends Dict<any>>(obj: T) => (keyof T)[];
/**
 * Object.entries polyfill for Nodev10 compatibility
 */
export declare const fromEntries: <T extends unknown>(entries: [string, any][]) => T;
/**
 * Get the CSS variable ref stored in the theme
 */
export declare const getCSSVar: (theme: Dict, scale: string, value: any) => any;
//# sourceMappingURL=object.d.ts.map