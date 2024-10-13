import { HTMLChakraProps, ThemingProps } from "@chakra-ui/system";
import * as React from "react";
export interface FormLabelProps extends HTMLChakraProps<"label">, ThemingProps<"FormLabel"> {
    /**
     * @type React.ReactElement
     */
    requiredIndicator?: React.ReactElement;
}
/**
 * Used to enhance the usability of form controls.
 *
 * It is used to inform users as to what information
 * is requested for a form field.
 *
 * ♿️ Accessibility: Every form field should have a form label.
 */
export declare const FormLabel: import("@chakra-ui/system").ComponentWithAs<"label", FormLabelProps>;
export interface RequiredIndicatorProps extends HTMLChakraProps<"span"> {
}
/**
 * Used to show a "required" text or an asterisks (*) to indicate that
 * a field is required.
 */
export declare const RequiredIndicator: import("@chakra-ui/system").ComponentWithAs<"span", RequiredIndicatorProps>;
//# sourceMappingURL=form-label.d.ts.map