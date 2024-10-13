/**
 * Creates a constant value over the lifecycle of a component.
 *
 * Even if `useMemo` is provided an empty array as its final argument, it doesn't offer
 * a guarantee that it won't re-run for performance reasons later on. By using `useConstant`
 * you can ensure that initialisers don't execute twice or more.
 */
export declare function useConst<T extends any | (() => any)>(init: T): T;
//# sourceMappingURL=use-const.d.ts.map