import { ChakraProps } from "@chakra-ui/system";
import { HTMLMotionProps } from "framer-motion";
import * as React from "react";
export interface ModalTransitionProps extends Omit<HTMLMotionProps<"section">, "color" | "transition">, ChakraProps {
    preset: "slideInBottom" | "slideInRight" | "scale" | "none";
}
export declare const ModalTransition: React.ForwardRefExoticComponent<ModalTransitionProps & React.RefAttributes<any>>;
//# sourceMappingURL=modal-transition.d.ts.map