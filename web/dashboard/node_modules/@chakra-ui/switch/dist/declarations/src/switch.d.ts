import { UseCheckboxProps } from "@chakra-ui/checkbox";
import { ThemingProps, HTMLChakraProps, SystemProps } from "@chakra-ui/system";
export interface SwitchProps extends Omit<UseCheckboxProps, "isIndeterminate">, Omit<HTMLChakraProps<"label">, keyof UseCheckboxProps>, ThemingProps<"Switch"> {
    /**
     * The spacing between the switch and its label text
     * @default 0.5rem
     * @type SystemProps["marginLeft"]
     */
    spacing?: SystemProps["marginLeft"];
}
export declare const Switch: import("@chakra-ui/system").ComponentWithAs<"input", SwitchProps>;
//# sourceMappingURL=switch.d.ts.map