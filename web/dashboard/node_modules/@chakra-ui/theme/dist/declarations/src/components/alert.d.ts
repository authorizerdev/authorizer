import type { PartsStyleFunction } from "@chakra-ui/theme-tools";
declare const _default: {
    parts: ("title" | "description" | "icon" | "container")[];
    baseStyle: Partial<Record<"title" | "description" | "icon" | "container", import("@chakra-ui/styled-system").CSSObject>>;
    variants: {
        subtle: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
        "left-accent": PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
        "top-accent": PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
        solid: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
    };
    defaultProps: {
        variant: string;
        colorScheme: string;
    };
};
export default _default;
//# sourceMappingURL=alert.d.ts.map