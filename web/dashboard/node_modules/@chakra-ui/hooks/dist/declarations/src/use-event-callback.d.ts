import * as React from "react";
/**
 * React hook for performant `useCallbacks`
 *
 * @see https://github.com/facebook/react/issues/14099#issuecomment-440013892
 *
 * @deprecated Use `useCallbackRef` instead. `useEventCallback` will be removed
 * in a future version.
 */
export declare function useEventCallback<E extends Event | React.SyntheticEvent>(callback: (event: E, ...args: any[]) => void): (event: E, ...args: any[]) => void;
//# sourceMappingURL=use-event-callback.d.ts.map