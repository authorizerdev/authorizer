import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import { PropGetter, PropGetterV2 } from "@chakra-ui/react-utils";
import * as React from "react";
export interface FormControlOptions {
    /**
     * If `true`, the form control will be required. This has 2 side effects:
     * - The `FormLabel` will show a required indicator
     * - The form element (e.g, Input) will have `aria-required` set to `true`
     */
    isRequired?: boolean;
    /**
     * If `true`, the form control will be disabled. This has 2 side effects:
     * - The `FormLabel` will have `data-disabled` attribute
     * - The form element (e.g, Input) will be disabled
     */
    isDisabled?: boolean;
    /**
     * If `true`, the form control will be invalid. This has 2 side effects:
     * - The `FormLabel` and `FormErrorIcon` will have `data-invalid` set to `true`
     * - The form element (e.g, Input) will have `aria-invalid` set to `true`
     */
    isInvalid?: boolean;
    /**
     * If `true`, the form control will be readonly
     */
    isReadOnly?: boolean;
}
interface FormControlContext extends FormControlOptions {
    /**
     * The label text used to inform users as to what information is
     * requested for a text field.
     */
    label?: string;
    /**
     * The custom `id` to use for the form control. This is passed directly to the form element (e.g, Input).
     * - The form element (e.g Input) gets the `id`
     * - The form label id: `form-label-${id}`
     * - The form error text id: `form-error-text-${id}`
     * - The form helper text id: `form-helper-text-${id}`
     */
    id?: string;
}
declare type FormControlProviderContext = Omit<ReturnType<typeof useFormControlProvider>, "getRootProps" | "htmlProps">;
declare const useFormControlContext: () => FormControlProviderContext;
export { useFormControlContext };
declare function useFormControlProvider(props: FormControlContext): {
    isRequired: boolean;
    isInvalid: boolean;
    isReadOnly: boolean;
    isDisabled: boolean;
    isFocused: boolean;
    onFocus: () => void;
    onBlur: () => void;
    hasFeedbackText: boolean;
    setHasFeedbackText: React.Dispatch<React.SetStateAction<boolean>>;
    hasHelpText: boolean;
    setHasHelpText: React.Dispatch<React.SetStateAction<boolean>>;
    id: string;
    labelId: string;
    feedbackId: string;
    helpTextId: string;
    htmlProps: {
        /**
         * The label text used to inform users as to what information is
         * requested for a text field.
         */
        label?: string | undefined;
    };
    getHelpTextProps: PropGetter<any, {}>;
    getErrorMessageProps: PropGetter<any, {}>;
    getRootProps: PropGetterV2<"div", {}>;
    getLabelProps: PropGetterV2<"label", {}>;
    getRequiredIndicatorProps: PropGetter<any, {}>;
};
export interface FormControlProps extends HTMLChakraProps<"div">, ThemingProps<"FormControl">, FormControlContext {
}
/**
 * FormControl provides context such as
 * `isInvalid`, `isDisabled`, and `isRequired` to form elements.
 *
 * This is commonly used in form elements such as `input`,
 * `select`, `textarea`, etc.
 */
export declare const FormControl: import("@chakra-ui/system").ComponentWithAs<"div", FormControlProps>;
export interface HelpTextProps extends HTMLChakraProps<"div"> {
}
/**
 * FormHelperText
 *
 * Assistive component that conveys additional guidance
 * about the field, such as how it will be used and what
 * types in values should be provided.
 */
export declare const FormHelperText: import("@chakra-ui/system").ComponentWithAs<"div", HelpTextProps>;
//# sourceMappingURL=form-control.d.ts.map