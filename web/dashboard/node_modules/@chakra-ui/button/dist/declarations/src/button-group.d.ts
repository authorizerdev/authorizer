import { SystemProps, ThemingProps, HTMLChakraProps } from "@chakra-ui/system";
export interface ButtonGroupProps extends HTMLChakraProps<"div">, ThemingProps<"Button"> {
    /**
     * If `true`, the borderRadius of button that are direct children will be altered
     * to look flushed together
     */
    isAttached?: boolean;
    /**
     * If `true`, all wrapped button will be disabled
     */
    isDisabled?: boolean;
    /**
     * The spacing between the buttons
     * @default '0.5rem'
     * @type SystemProps["marginRight"]
     */
    spacing?: SystemProps["marginRight"];
}
interface ButtonGroupContext extends ThemingProps<"ButtonGroup"> {
    isDisabled?: boolean;
}
declare const useButtonGroup: () => ButtonGroupContext;
export { useButtonGroup };
export declare const ButtonGroup: import("@chakra-ui/system").ComponentWithAs<"div", ButtonGroupProps>;
//# sourceMappingURL=button-group.d.ts.map