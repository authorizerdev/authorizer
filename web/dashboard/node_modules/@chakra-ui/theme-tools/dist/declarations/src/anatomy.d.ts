/**
 * Used to define the anatomy/parts of a component in a way that provides
 * a consistent API for `className`, css selector and `theming`.
 */
export declare class Anatomy<T extends string = string> {
    private name;
    private map;
    private called;
    constructor(name: string);
    /**
     * Prevents user from calling `.parts` multiple times.
     * It should only be called once.
     */
    private assert;
    /**
     * Add the core parts of the components
     */
    parts: <V extends string>(...values: V[]) => Omit<Anatomy<V>, "parts">;
    /**
     * Extend the component anatomy to includes new parts
     */
    extend: <U extends string>(...parts: U[]) => Omit<Anatomy<T | U>, "parts">;
    /**
     * Get all selectors for the component anatomy
     */
    get selectors(): Record<T, string>;
    /**
     * Get all classNames for the component anatomy
     */
    get classNames(): Record<T, string>;
    /**
     * Get all parts as array of string
     */
    get keys(): T[];
    /**
     * Creates the part object for the given part
     */
    toPart: (part: string) => {
        className: string;
        selector: string;
        toString: () => string;
    } & string;
    /**
     * Used to get the derived type of the anatomy
     */
    __type: T;
}
export declare function anatomy(name: string): Anatomy<string>;
//# sourceMappingURL=anatomy.d.ts.map