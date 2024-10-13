import type { AlertStatus } from "@chakra-ui/alert";
import { ColorMode, useChakra } from "@chakra-ui/system";
import * as React from "react";
import { ToastPositionWithLogical } from "./toast.placement";
import type { RenderProps, ToastId, ToastOptions } from "./toast.types";
export interface UseToastOptions {
    /**
     * The placement of the toast
     *
     * @default "bottom"
     */
    position?: ToastPositionWithLogical;
    /**
     * The delay before the toast hides (in milliseconds)
     * If set to `null`, toast will never dismiss.
     *
     * @default 5000 ( = 5000ms )
     */
    duration?: ToastOptions["duration"];
    /**
     * Render a component toast component.
     * Any component passed will receive 2 props: `id` and `onClose`.
     */
    render?(props: RenderProps): React.ReactNode;
    /**
     * The title of the toast
     */
    title?: React.ReactNode;
    /**
     * The description of the toast
     */
    description?: React.ReactNode;
    /**
     * If `true`, toast will show a close button
     */
    isClosable?: boolean;
    /**
     * The alert component `variant` to use
     */
    variant?: "subtle" | "solid" | "left-accent" | "top-accent" | (string & {});
    /**
     * The status of the toast.
     */
    status?: AlertStatus;
    /**
     * The `id` of the toast.
     *
     * Mostly used when you need to prevent duplicate.
     * By default, we generate a unique `id` for each toast
     */
    id?: ToastId;
    /**
     * Callback function to run side effects after the toast has closed.
     */
    onCloseComplete?: () => void;
    /**
     * Optional style overrides for the container wrapping the toast component.
     */
    containerStyle?: React.CSSProperties;
}
export declare type IToast = UseToastOptions;
export declare type CreateStandAloneToastParam = Partial<ReturnType<typeof useChakra> & {
    setColorMode: (value: ColorMode) => void;
    defaultOptions: UseToastOptions;
}>;
export declare const defaultStandaloneParam: Required<CreateStandAloneToastParam>;
/**
 * Create a toast from outside of React Components
 */
export declare function createStandaloneToast({ theme, colorMode, toggleColorMode, setColorMode, defaultOptions, }?: CreateStandAloneToastParam): {
    (options?: UseToastOptions | undefined): ToastId | undefined;
    close: (id: ToastId) => void;
    closeAll: (options?: import("./toast.types").CloseAllToastsOptions | undefined) => void;
    update(id: ToastId, options: Omit<UseToastOptions, "id">): void;
    isActive: (id: ToastId) => boolean | undefined;
};
/**
 * React hook used to create a function that can be used
 * to show toasts in an application.
 */
export declare function useToast(options?: UseToastOptions): {
    (options?: UseToastOptions | undefined): ToastId | undefined;
    close: (id: ToastId) => void;
    closeAll: (options?: import("./toast.types").CloseAllToastsOptions | undefined) => void;
    update(id: ToastId, options: Omit<UseToastOptions, "id">): void;
    isActive: (id: ToastId) => boolean | undefined;
};
export default useToast;
//# sourceMappingURL=use-toast.d.ts.map