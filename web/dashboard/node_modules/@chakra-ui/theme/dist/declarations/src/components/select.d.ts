import type { PartsStyleFunction } from "@chakra-ui/theme-tools";
declare const _default: {
    parts: ("icon" | "field")[];
    baseStyle: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"icon" | "field">, "parts">>;
    sizes: Record<string, Partial<Record<"icon" | "field", import("@chakra-ui/styled-system").CSSObject>>>;
    variants: {
        outline: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
        filled: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
        flushed: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
        unstyled: Partial<Record<"element" | "field" | "addon", import("@chakra-ui/styled-system").CSSObject>>;
    };
    defaultProps: {
        size: string;
        variant: string;
    };
};
export default _default;
//# sourceMappingURL=select.d.ts.map