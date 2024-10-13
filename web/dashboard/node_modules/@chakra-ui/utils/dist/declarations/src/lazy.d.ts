export declare type LazyBehavior = "unmount" | "keepMounted";
interface DetermineLazyBehaviorOptions {
    hasBeenSelected?: boolean;
    isLazy?: boolean;
    isSelected?: boolean;
    lazyBehavior?: LazyBehavior;
}
/**
 * Determines whether the children of a disclosure widget
 * should be rendered or not, depending on the lazy behavior.
 *
 * Used in accordion, tabs, popover, menu and other disclosure
 * widgets.
 */
export declare function determineLazyBehavior(options: DetermineLazyBehaviorOptions): boolean;
export {};
//# sourceMappingURL=lazy.d.ts.map