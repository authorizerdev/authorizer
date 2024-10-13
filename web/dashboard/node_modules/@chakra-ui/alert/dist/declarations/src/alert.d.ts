import { ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
declare const STATUSES: {
    info: {
        icon: React.FC<import("@chakra-ui/icon").IconProps>;
        colorScheme: string;
    };
    warning: {
        icon: React.FC<import("@chakra-ui/icon").IconProps>;
        colorScheme: string;
    };
    success: {
        icon: React.FC<import("@chakra-ui/icon").IconProps>;
        colorScheme: string;
    };
    error: {
        icon: React.FC<import("@chakra-ui/icon").IconProps>;
        colorScheme: string;
    };
};
export declare type AlertStatus = keyof typeof STATUSES;
interface AlertOptions {
    /**
     * The status of the alert
     */
    status?: AlertStatus;
}
export interface AlertProps extends HTMLChakraProps<"div">, AlertOptions, ThemingProps<"Alert"> {
}
/**
 * Alert is used to communicate the state or status of a
 * page, feature or action
 */
export declare const Alert: import("@chakra-ui/system").ComponentWithAs<"div", AlertProps>;
export interface AlertTitleProps extends HTMLChakraProps<"div"> {
}
export declare const AlertTitle: import("@chakra-ui/system").ComponentWithAs<"div", AlertTitleProps>;
export interface AlertDescriptionProps extends HTMLChakraProps<"div"> {
}
export declare const AlertDescription: import("@chakra-ui/system").ComponentWithAs<"div", AlertDescriptionProps>;
export interface AlertIconProps extends HTMLChakraProps<"span"> {
}
export declare const AlertIcon: React.FC<AlertIconProps>;
export {};
//# sourceMappingURL=alert.d.ts.map