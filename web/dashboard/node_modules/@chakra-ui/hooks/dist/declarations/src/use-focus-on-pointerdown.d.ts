import { RefObject } from "react";
export interface UseFocusOnMouseDownProps {
    enabled?: boolean;
    ref: RefObject<HTMLElement>;
    elements?: Array<RefObject<HTMLElement> | HTMLElement | null>;
}
/**
 * Polyfill to get `relatedTarget` working correctly consistently
 * across all browsers.
 *
 * It ensures that elements receives focus on pointer down if
 * it's not the active active element.
 *
 * @internal
 */
export declare function useFocusOnPointerDown(props: UseFocusOnMouseDownProps): void;
//# sourceMappingURL=use-focus-on-pointerdown.d.ts.map