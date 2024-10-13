import type { CloseAllToastsOptions, ToastId, ToastMessage, ToastOptions } from "./toast.types";
declare class Toaster {
    private createToast?;
    private removeAll?;
    private closeToast?;
    private updateToast?;
    private isToastActive?;
    /**
     * Initialize the manager and mount it in the DOM
     * inside the portal node.
     *
     * @todo
     *
     * Update toast constructor to use `PortalManager`'s node or document.body.
     * Once done, we can remove the `zIndex` in `toast.manager.tsx`
     */
    constructor();
    private bindFunctions;
    notify: (message: ToastMessage, options?: Partial<ToastOptions>) => ToastId | undefined;
    close: (id: ToastId) => void;
    closeAll: (options?: CloseAllToastsOptions | undefined) => void;
    update: (id: ToastId, options?: Partial<ToastOptions>) => void;
    isActive: (id: ToastId) => boolean | undefined;
}
export declare const toast: Toaster;
export {};
//# sourceMappingURL=toast.class.d.ts.map