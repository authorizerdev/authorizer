interface FocusNextOptions {
    scope?: HTMLElement | HTMLDocument;
    cycle?: boolean;
    focusOptions?: FocusOptions;
}
export declare const focusNextElement: (baseElement: Element, options?: FocusNextOptions) => void;
export declare const focusPrevElement: (baseElement: Element, options?: FocusNextOptions) => void;
export {};
