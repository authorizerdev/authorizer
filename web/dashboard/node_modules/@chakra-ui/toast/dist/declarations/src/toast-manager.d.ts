import * as React from "react";
import { ToastPosition } from "./toast.placement";
import type { CloseAllToastsOptions, ToastId, ToastMessage, ToastOptions, ToastState } from "./toast.types";
export interface ToastMethods {
    notify: (message: ToastMessage, options: CreateToastOptions) => ToastId;
    closeAll: (options?: CloseAllToastsOptions) => void;
    close: (id: ToastId) => void;
    update: (id: ToastId, options: CreateToastOptions) => void;
    isActive: (id: ToastId) => boolean;
}
interface Props {
    notify: (methods: ToastMethods) => void;
}
declare type CreateToastOptions = Partial<Pick<ToastOptions, "status" | "duration" | "position" | "id" | "onCloseComplete" | "containerStyle">>;
/**
 * Manages the creation, and removal of toasts
 * across all corners ("top", "bottom", etc.)
 */
export declare class ToastManager extends React.Component<Props, ToastState> {
    /**
     * Static id counter to create unique ids
     * for each toast
     */
    static counter: number;
    /**
     * State to track all the toast across all positions
     */
    state: ToastState;
    constructor(props: Props);
    /**
     * Function to actually create a toast and add it
     * to state at the specified position
     */
    notify: (message: ToastMessage, options: CreateToastOptions) => ToastId;
    /**
     * Update a specific toast with new options based on the
     * passed `id`
     */
    updateToast: (id: ToastId, options: CreateToastOptions) => void;
    /**
     * Close all toasts at once.
     * If given positions, will only close those.
     */
    closeAll: ({ positions }?: CloseAllToastsOptions) => void;
    /**
     * Create properties for a new toast
     */
    createToast: (message: ToastMessage, options: CreateToastOptions) => {
        id: ToastId;
        message: ToastMessage;
        position: ToastPosition;
        duration: number | null | undefined;
        onCloseComplete: (() => void) | undefined;
        onRequestRemove: () => void;
        status: import("./toast.types").Status | undefined;
        requestClose: boolean;
        containerStyle: React.CSSProperties | undefined;
    };
    /**
     * Requests to close a toast based on its id and position
     */
    closeToast: (id: ToastId) => void;
    /**
     * Delete a toast record at its position
     */
    removeToast: (id: ToastId, position: ToastPosition) => void;
    isVisible: (id: ToastId) => boolean;
    /**
     * Compute the style of a toast based on its position
     */
    getStyle: (position: ToastPosition) => React.CSSProperties;
    render(): JSX.Element[];
}
export {};
//# sourceMappingURL=toast-manager.d.ts.map