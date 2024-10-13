import { CloseButtonProps } from "@chakra-ui/close-button";
import { FocusLockProps } from "@chakra-ui/focus-lock";
import { PortalProps } from "@chakra-ui/portal";
import { ChakraProps, HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import { FocusableElement } from "@chakra-ui/utils";
import { HTMLMotionProps } from "framer-motion";
import * as React from "react";
import { UseModalProps, UseModalReturn } from "./use-modal";
interface ModalOptions extends Pick<FocusLockProps, "lockFocusAcrossFrames"> {
    /**
     * If `false`, focus lock will be disabled completely.
     *
     * This is useful in situations where you still need to interact with
     * other surrounding elements.
     *
     * 🚨Warning: We don't recommend doing this because it hurts the
     * accessibility of the modal, based on WAI-ARIA specifications.
     *
     * @default true
     */
    trapFocus?: boolean;
    /**
     * If `true`, the modal will autofocus the first enabled and interactive
     * element within the `ModalContent`
     *
     * @default true
     */
    autoFocus?: boolean;
    /**
     * The `ref` of element to receive focus when the modal opens.
     */
    initialFocusRef?: React.RefObject<FocusableElement>;
    /**
     * The `ref` of element to receive focus when the modal closes.
     */
    finalFocusRef?: React.RefObject<FocusableElement>;
    /**
     * If `true`, the modal will return focus to the element that triggered it when it closes.
     * @default true
     */
    returnFocusOnClose?: boolean;
    /**
     * If `true`, scrolling will be disabled on the `body` when the modal opens.
     *  @default true
     */
    blockScrollOnMount?: boolean;
    /**
     * Handle zoom/pinch gestures on iOS devices when scroll locking is enabled.
     * Defaults to `false`.
     */
    allowPinchZoom?: boolean;
    /**
     * If `true`, a `padding-right` will be applied to the body element
     * that's equal to the width of the scrollbar.
     *
     * This can help prevent some unpleasant flickering effect
     * and content adjustment when the modal opens
     */
    preserveScrollBarGap?: boolean;
}
declare type ScrollBehavior = "inside" | "outside";
declare type MotionPreset = "slideInBottom" | "slideInRight" | "scale" | "none";
export interface ModalProps extends UseModalProps, ModalOptions, ThemingProps<"Modal"> {
    children: React.ReactNode;
    /**
     *  If `true`, the modal will be centered on screen.
     * @default false
     */
    isCentered?: boolean;
    /**
     * Where scroll behavior should originate.
     * - If set to `inside`, scroll only occurs within the `ModalBody`.
     * - If set to `outside`, the entire `ModalContent` will scroll within the viewport.
     *
     * @default "outside"
     */
    scrollBehavior?: ScrollBehavior;
    /**
     * Props to be forwarded to the portal component
     */
    portalProps?: Pick<PortalProps, "appendToParentPortal" | "containerRef">;
    /**
     * The transition that should be used for the modal
     */
    motionPreset?: MotionPreset;
}
interface ModalContext extends ModalOptions, UseModalReturn {
    /**
     * The transition that should be used for the modal
     */
    motionPreset?: MotionPreset;
}
declare const ModalContextProvider: React.Provider<ModalContext>, useModalContext: () => ModalContext;
export { ModalContextProvider, useModalContext };
/**
 * Modal provides context, theming, and accessibility properties
 * to all other modal components.
 *
 * It doesn't render any DOM node.
 */
export declare const Modal: React.FC<ModalProps>;
export interface ModalContentProps extends HTMLChakraProps<"section"> {
    /**
     * The props to forward to the modal's content wrapper
     */
    containerProps?: HTMLChakraProps<"div">;
}
/**
 * ModalContent is used to group modal's content. It has all the
 * necessary `aria-*` properties to indicate that it is a modal
 */
export declare const ModalContent: import("@chakra-ui/system").ComponentWithAs<"section", ModalContentProps>;
interface ModalFocusScopeProps {
    /**
     * @type React.ReactElement
     */
    children: React.ReactElement;
}
export declare function ModalFocusScope(props: ModalFocusScopeProps): JSX.Element;
export interface ModalOverlayProps extends Omit<HTMLMotionProps<"div">, "color" | "transition">, ChakraProps {
    children?: React.ReactNode;
}
/**
 * ModalOverlay renders a backdrop behind the modal. It is
 * also used as a wrapper for the modal content for better positioning.
 *
 * @see Docs https://chakra-ui.com/modal
 */
export declare const ModalOverlay: import("@chakra-ui/system").ComponentWithAs<"div", ModalOverlayProps>;
export interface ModalHeaderProps extends HTMLChakraProps<"header"> {
}
/**
 * ModalHeader
 *
 * React component that houses the title of the modal.
 *
 * @see Docs https://chakra-ui.com/modal
 */
export declare const ModalHeader: import("@chakra-ui/system").ComponentWithAs<"header", ModalHeaderProps>;
export interface ModalBodyProps extends HTMLChakraProps<"div"> {
}
/**
 * ModalBody
 *
 * React component that houses the main content of the modal.
 *
 * @see Docs https://chakra-ui.com/modal
 */
export declare const ModalBody: import("@chakra-ui/system").ComponentWithAs<"div", ModalBodyProps>;
export interface ModalFooterProps extends HTMLChakraProps<"footer"> {
}
/**
 * ModalFooter houses the action buttons of the modal.
 * @see Docs https://chakra-ui.com/modal
 */
export declare const ModalFooter: import("@chakra-ui/system").ComponentWithAs<"footer", ModalFooterProps>;
/**
 * ModalCloseButton is used closes the modal.
 *
 * You don't need to pass the `onClick` to it, it reads the
 * `onClose` action from the modal context.
 */
export declare const ModalCloseButton: import("@chakra-ui/system").ComponentWithAs<"button", CloseButtonProps>;
//# sourceMappingURL=modal.d.ts.map