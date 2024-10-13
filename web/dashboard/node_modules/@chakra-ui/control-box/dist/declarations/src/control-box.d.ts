import { SystemStyleObject, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
export interface ControlBoxOptions {
    type?: "checkbox" | "radio";
    _hover?: SystemStyleObject;
    _invalid?: SystemStyleObject;
    _disabled?: SystemStyleObject;
    _focus?: SystemStyleObject;
    _checked?: SystemStyleObject;
    _child?: SystemStyleObject;
    _checkedAndChild?: SystemStyleObject;
    _checkedAndDisabled?: SystemStyleObject;
    _checkedAndFocus?: SystemStyleObject;
    _checkedAndHover?: SystemStyleObject;
}
export declare type IControlBox = ControlBoxOptions;
interface BaseControlProps extends Omit<HTMLChakraProps<"div">, keyof ControlBoxOptions> {
}
export interface ControlBoxProps extends BaseControlProps, ControlBoxOptions {
}
export declare const ControlBox: React.FC<ControlBoxProps>;
export default ControlBox;
//# sourceMappingURL=control-box.d.ts.map