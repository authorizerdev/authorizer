export declare const hasDisplayNone: (element: HTMLElement) => boolean;
export declare const hasTabIndex: (element: HTMLElement) => boolean;
export declare const hasNegativeTabIndex: (element: HTMLElement) => boolean;
export declare function isDisabled(element: HTMLElement): boolean;
export interface FocusableElement {
    focus(options?: FocusOptions): void;
}
export declare function isInputElement(element: FocusableElement): element is HTMLInputElement;
export declare function isActiveElement(element: FocusableElement): boolean;
export declare function hasFocusWithin(element: HTMLElement): boolean;
export declare function isHidden(element: HTMLElement): boolean;
export declare function isContentEditable(element: HTMLElement): boolean;
export declare function isFocusable(element: HTMLElement): boolean;
export declare function isTabbable(element?: HTMLElement | null): boolean;
//# sourceMappingURL=tabbable.d.ts.map