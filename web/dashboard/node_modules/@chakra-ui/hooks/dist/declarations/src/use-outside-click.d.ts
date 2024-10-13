import { RefObject } from "react";
export interface UseOutsideClickProps {
    /**
     * Whether the hook is enabled
     */
    enabled?: boolean;
    /**
     * The reference to a DOM element.
     */
    ref: RefObject<HTMLElement>;
    /**
     * Function invoked when a click is triggered outside the referenced element.
     */
    handler?: (e: Event) => void;
}
/**
 * Example, used in components like Dialogs and Popovers so they can close
 * when a user clicks outside them.
 */
export declare function useOutsideClick(props: UseOutsideClickProps): void;
//# sourceMappingURL=use-outside-click.d.ts.map