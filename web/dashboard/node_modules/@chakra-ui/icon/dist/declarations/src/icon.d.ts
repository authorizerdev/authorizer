import { ChakraProps } from "@chakra-ui/system";
import * as React from "react";
export interface IconProps extends Omit<React.SVGAttributes<SVGElement>, keyof ChakraProps>, ChakraProps {
}
export declare const Icon: import("@chakra-ui/system").ComponentWithAs<"svg", IconProps>;
export default Icon;
//# sourceMappingURL=icon.d.ts.map