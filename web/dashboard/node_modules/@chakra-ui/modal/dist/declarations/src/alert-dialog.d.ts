/// <reference types="react" />
import { ModalContentProps, ModalProps } from "./modal";
export interface AlertDialogProps extends Omit<ModalProps, "initialFocusRef"> {
    leastDestructiveRef: ModalProps["initialFocusRef"];
}
export declare function AlertDialog(props: AlertDialogProps): JSX.Element;
export declare const AlertDialogContent: import("@chakra-ui/system").ComponentWithAs<"section", ModalContentProps>;
export { ModalBody as AlertDialogBody, ModalCloseButton as AlertDialogCloseButton, ModalFooter as AlertDialogFooter, ModalHeader as AlertDialogHeader, ModalOverlay as AlertDialogOverlay, } from "./modal";
//# sourceMappingURL=alert-dialog.d.ts.map