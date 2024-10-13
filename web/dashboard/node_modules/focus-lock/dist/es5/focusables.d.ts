interface FocusableIn {
    node: HTMLElement;
    index: number;
    lockItem: boolean;
    guard: boolean;
}
export declare const getFocusabledIn: (topNode: HTMLElement) => FocusableIn[];
export {};
