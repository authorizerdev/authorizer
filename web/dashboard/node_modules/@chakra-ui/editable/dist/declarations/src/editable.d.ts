import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import { MaybeRenderProp } from "@chakra-ui/react-utils";
import { UseEditableProps, UseEditableReturn } from "./use-editable";
declare type EditableContext = Omit<UseEditableReturn, "htmlProps">;
declare type RenderProps = Pick<UseEditableReturn, "isEditing" | "onSubmit" | "onCancel" | "onEdit">;
interface BaseEditableProps extends Omit<HTMLChakraProps<"div">, "onChange" | "value" | "defaultValue" | "onSubmit"> {
}
export interface EditableProps extends UseEditableProps, BaseEditableProps, ThemingProps<"Editable"> {
    children?: MaybeRenderProp<RenderProps>;
}
/**
 * Editable
 *
 * The wrapper that provides context and logic for all editable
 * components. It renders a `div`
 */
export declare const Editable: import("@chakra-ui/system").ComponentWithAs<"div", EditableProps>;
export interface EditablePreviewProps extends HTMLChakraProps<"div"> {
}
/**
 * EditablePreview
 *
 * The `span` used to display the final value, in the `preview` mode
 */
export declare const EditablePreview: import("@chakra-ui/system").ComponentWithAs<"span", EditablePreviewProps>;
export interface EditableInputProps extends HTMLChakraProps<"input"> {
}
/**
 * EditableInput
 *
 * The input used in the `edit` mode
 */
export declare const EditableInput: import("@chakra-ui/system").ComponentWithAs<"input", EditableInputProps>;
/**
 * React hook use to gain access to the editable state and actions.
 */
export declare function useEditableState(): {
    isEditing: boolean;
    onSubmit: () => void;
    onCancel: () => void;
    onEdit: () => void;
    isDisabled: boolean | undefined;
};
/**
 * React hook use to create controls for the editable component
 */
export declare function useEditableControls(): Pick<EditableContext, "isEditing" | "getEditButtonProps" | "getCancelButtonProps" | "getSubmitButtonProps">;
export {};
//# sourceMappingURL=editable.d.ts.map