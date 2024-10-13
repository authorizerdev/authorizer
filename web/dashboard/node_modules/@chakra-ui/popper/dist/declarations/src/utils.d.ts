import { Placement } from "@popperjs/core";
export declare const cssVars: {
    readonly arrowShadowColor: {
        var: string;
        varRef: string;
    };
    readonly arrowSize: {
        var: string;
        varRef: string;
    };
    readonly arrowSizeHalf: {
        var: string;
        varRef: string;
    };
    readonly arrowBg: {
        var: string;
        varRef: string;
    };
    readonly transformOrigin: {
        var: string;
        varRef: string;
    };
    readonly arrowOffset: {
        var: string;
        varRef: string;
    };
};
export declare function getBoxShadow(placement: Placement): "1px 1px 1px 0 var(--popper-arrow-shadow-color)" | "-1px -1px 1px 0 var(--popper-arrow-shadow-color)" | "-1px 1px 1px 0 var(--popper-arrow-shadow-color)" | "1px -1px 1px 0 var(--popper-arrow-shadow-color)" | undefined;
export declare const toTransformOrigin: (placement: Placement) => any;
declare const defaultEventListeners: {
    scroll: boolean;
    resize: boolean;
};
export declare function getEventListenerOptions(value?: boolean | Partial<typeof defaultEventListeners>): {
    enabled?: boolean | undefined;
    options?: {
        scroll: boolean;
        resize: boolean;
    } | undefined;
};
export {};
//# sourceMappingURL=utils.d.ts.map