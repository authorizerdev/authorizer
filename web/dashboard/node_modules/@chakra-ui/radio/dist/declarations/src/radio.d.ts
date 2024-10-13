import { SystemProps, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
import { UseRadioProps } from "./use-radio";
declare type Omitted = "onChange" | "defaultChecked" | "checked";
interface BaseControlProps extends Omit<HTMLChakraProps<"div">, Omitted> {
}
export interface RadioProps extends UseRadioProps, ThemingProps<"Radio">, BaseControlProps {
    /**
     * The spacing between the checkbox and its label text
     * @default 0.5rem
     * @type SystemProps["marginLeft"]
     */
    spacing?: SystemProps["marginLeft"];
    /**
     * If `true`, the radio will occupy the full width of its parent container
     *
     * @deprecated
     * This component defaults to 100% width,
     * please use the props `maxWidth` or `width` to configure
     */
    isFullWidth?: boolean;
}
/**
 * Radio component is used in forms when a user needs to select a single value from
 * several options.
 *
 * @see Docs https://chakra-ui.com/radio
 */
export declare const Radio: import("@chakra-ui/system").ComponentWithAs<"input", RadioProps>;
export {};
//# sourceMappingURL=radio.d.ts.map