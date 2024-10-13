import { UnionStringArray } from "@chakra-ui/utils";
import { ThemingProps } from "./system.types";
/**
 * Carefully selected html elements for chakra components.
 * This is mostly for `chakra.<element>` syntax.
 */
export declare const domElements: readonly ["a", "b", "article", "aside", "blockquote", "button", "caption", "cite", "circle", "code", "dd", "div", "dl", "dt", "fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr", "img", "input", "kbd", "label", "li", "main", "mark", "nav", "ol", "p", "path", "pre", "q", "rect", "s", "svg", "section", "select", "strong", "small", "span", "sub", "sup", "table", "tbody", "td", "textarea", "tfoot", "th", "thead", "tr", "ul"];
export declare type DOMElements = UnionStringArray<typeof domElements>;
export declare function omitThemingProps<T extends ThemingProps>(props: T): import("@chakra-ui/utils").Omit<T, "colorScheme" | "variant" | "size" | "styleConfig">;
export default function isTag(target: any): boolean;
export declare function getDisplayName(primitive: any): string;
//# sourceMappingURL=system.utils.d.ts.map