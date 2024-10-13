import { Dict } from "@chakra-ui/utils";
/**
 * Expands an array or object syntax responsive style.
 *
 * @example
 * expandResponsive({ mx: [1, 2] })
 * // or
 * expandResponsive({ mx: { base: 1, sm: 2 } })
 *
 * // => { mx: 1, "@media(min-width:<sm>)": { mx: 2 } }
 */
export declare const expandResponsive: (styles: Dict) => (theme: Dict) => Dict<any>;
//# sourceMappingURL=expand-responsive.d.ts.map