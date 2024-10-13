import { IconProps } from "@chakra-ui/icon";
import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
export interface StatLabelProps extends HTMLChakraProps<"dt"> {
}
export declare const StatLabel: import("@chakra-ui/system").ComponentWithAs<"dt", StatLabelProps>;
export interface StatHelpTextProps extends HTMLChakraProps<"dd"> {
}
export declare const StatHelpText: import("@chakra-ui/system").ComponentWithAs<"dd", StatHelpTextProps>;
export interface StatNumberProps extends HTMLChakraProps<"dd"> {
}
export declare const StatNumber: import("@chakra-ui/system").ComponentWithAs<"dd", StatNumberProps>;
export declare const StatDownArrow: React.FC<IconProps>;
export declare const StatUpArrow: React.FC<IconProps>;
export interface StatArrowProps extends IconProps {
    type?: "increase" | "decrease";
}
export declare const StatArrow: React.FC<StatArrowProps>;
export interface StatProps extends HTMLChakraProps<"div">, ThemingProps<"Stat"> {
}
export declare const Stat: import("@chakra-ui/system").ComponentWithAs<"div", StatProps>;
export interface StatGroupProps extends HTMLChakraProps<"div"> {
}
export declare const StatGroup: import("@chakra-ui/system").ComponentWithAs<"div", StatGroupProps>;
//# sourceMappingURL=stat.d.ts.map