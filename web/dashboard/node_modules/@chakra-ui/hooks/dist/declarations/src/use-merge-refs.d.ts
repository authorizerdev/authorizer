import * as React from "react";
declare type ReactRef<T> = React.Ref<T> | React.MutableRefObject<T>;
export declare function assignRef<T = any>(ref: ReactRef<T> | undefined, value: T): void;
/**
 * React hook that merges react refs into a single memoized function
 *
 * @example
 * import React from "react";
 * import { useMergeRefs } from `@chakra-ui/hooks`;
 *
 * const Component = React.forwardRef((props, ref) => {
 *   const internalRef = React.useRef();
 *   return <div {...props} ref={useMergeRefs(internalRef, ref)} />;
 * });
 */
export declare function useMergeRefs<T>(...refs: (ReactRef<T> | undefined)[]): ((node: T) => void) | null;
export {};
//# sourceMappingURL=use-merge-refs.d.ts.map