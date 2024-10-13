import { Dict } from "./types";
export declare function isNumber(value: any): value is number;
export declare function isNotNumber(value: any): boolean;
export declare function isNumeric(value: any): boolean;
export declare function isArray<T>(value: any): value is Array<T>;
export declare function isEmptyArray(value: any): boolean;
export declare function isFunction<T extends Function = Function>(value: any): value is T;
export declare function isDefined(value: any): boolean;
export declare function isUndefined(value: any): value is undefined;
export declare function isObject(value: any): value is Dict;
export declare function isEmptyObject(value: any): boolean;
export declare function isNotEmptyObject(value: any): value is object;
export declare function isNull(value: any): value is null;
export declare function isString(value: any): value is string;
export declare function isCssVar(value: string): boolean;
export declare function isEmpty(value: any): boolean;
export declare const __DEV__: boolean;
export declare const __TEST__: boolean;
export declare function isRefObject(val: any): val is {
    current: any;
};
export declare function isInputEvent(value: any): value is {
    target: HTMLInputElement;
};
//# sourceMappingURL=assertion.d.ts.map