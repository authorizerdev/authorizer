import type { PartsStyleFunction, PartsStyleInterpolation } from "@chakra-ui/theme-tools";
declare const _default: {
    parts: ("tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator")[];
    baseStyle: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator">, "parts">>;
    sizes: Record<string, Partial<Record<"tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator", import("@chakra-ui/styled-system").CSSObject>>>;
    variants: Record<string, PartsStyleInterpolation<Omit<import("@chakra-ui/theme-tools").Anatomy<"tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator">, "parts">>>;
    defaultProps: {
        size: string;
        variant: string;
        colorScheme: string;
    };
};
export default _default;
//# sourceMappingURL=tabs.d.ts.map