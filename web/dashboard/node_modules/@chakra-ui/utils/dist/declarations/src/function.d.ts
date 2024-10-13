import { AnyFunction, FunctionArguments } from "./types";
export declare function runIfFn<T, U>(valueOrFn: T | ((...fnArgs: U[]) => T), ...args: U[]): T;
export declare function callAllHandlers<T extends (event: any) => void>(...fns: (T | undefined)[]): (event: FunctionArguments<T>[0]) => void;
export declare function callAll<T extends AnyFunction>(...fns: (T | undefined)[]): (arg: FunctionArguments<T>[0]) => void;
export declare const compose: <T>(fn1: (...args: T[]) => T, ...fns: ((...args: T[]) => T)[]) => (...args: T[]) => T;
export declare function once<T extends AnyFunction>(fn?: T | null): (this: any, ...args: Parameters<T>) => any;
export declare const noop: () => void;
declare type MessageOptions = {
    condition: boolean;
    message: string;
};
export declare const warn: (this: any, options: MessageOptions) => any;
export declare const error: (this: any, options: MessageOptions) => any;
export declare const pipe: <R>(...fns: ((a: R) => R)[]) => (v: R) => R;
declare type Point = {
    x: number;
    y: number;
};
export declare function distance<P extends Point | number>(a: P, b: P): number;
export {};
//# sourceMappingURL=function.d.ts.map