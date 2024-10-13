export declare type Merge<T, P> = P & Omit<T, keyof P>;
export declare type UnionStringArray<T extends Readonly<string[]>> = T[number];
export declare type Omit<T, K> = Pick<T, Exclude<keyof T, K>>;
export declare type LiteralUnion<T extends U, U extends any = string> = T | (U & {
    _?: never;
});
export declare type AnyFunction<T = any> = (...args: T[]) => any;
export declare type FunctionArguments<T extends Function> = T extends (...args: infer R) => any ? R : never;
export declare type Dict<T = any> = Record<string, T>;
export declare type Booleanish = boolean | "true" | "false";
export declare type StringOrNumber = string | number;
export declare type EventKeys = "ArrowDown" | "ArrowUp" | "ArrowLeft" | "ArrowRight" | "Enter" | "Space" | "Tab" | "Backspace" | "Control" | "Meta" | "Home" | "End" | "PageDown" | "PageUp" | "Delete" | "Escape" | " " | "Shift";
//# sourceMappingURL=types.d.ts.map