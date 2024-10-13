import type { PartsStyleFunction } from "@chakra-ui/theme-tools";
declare const _default: {
    parts: ("caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr")[];
    baseStyle: Partial<Record<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr", import("@chakra-ui/styled-system").CSSObject>>;
    variants: {
        simple: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr">, "parts">>;
        striped: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr">, "parts">>;
        unstyled: {};
    };
    sizes: Record<string, Partial<Record<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr", import("@chakra-ui/styled-system").CSSObject>>>;
    defaultProps: {
        variant: string;
        size: string;
        colorScheme: string;
    };
};
export default _default;
//# sourceMappingURL=table.d.ts.map