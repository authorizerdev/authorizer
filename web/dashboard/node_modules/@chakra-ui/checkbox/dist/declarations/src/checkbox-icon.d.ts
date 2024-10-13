import { chakra, PropsOf } from "@chakra-ui/system";
import { CustomDomComponent } from "framer-motion";
import * as React from "react";
declare const MotionSvg: CustomDomComponent<PropsOf<typeof chakra.svg>>;
export interface CheckboxIconProps extends PropsOf<typeof MotionSvg> {
    isIndeterminate?: boolean;
    isChecked?: boolean;
}
/**
 * CheckboxIcon is used to visually indicate the checked or indeterminate
 * state of a checkbox.
 *
 * @todo allow users pass their own icon svgs
 */
export declare const CheckboxIcon: React.FC<CheckboxIconProps>;
export {};
//# sourceMappingURL=checkbox-icon.d.ts.map