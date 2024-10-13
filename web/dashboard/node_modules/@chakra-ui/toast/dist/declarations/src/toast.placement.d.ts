export declare type LogicalToastPosition = "top-start" | "top-end" | "bottom-start" | "bottom-end";
export declare type ToastPositionWithLogical = LogicalToastPosition | "top" | "top-left" | "top-right" | "bottom" | "bottom-left" | "bottom-right";
export declare type ToastPosition = Exclude<ToastPositionWithLogical, LogicalToastPosition>;
export declare type WithoutLogicalPosition<T> = Omit<T, "position"> & {
    position?: ToastPosition;
};
export declare function getToastPlacement(position: ToastPositionWithLogical | undefined, dir: "ltr" | "rtl"): ToastPosition | undefined;
//# sourceMappingURL=toast.placement.d.ts.map