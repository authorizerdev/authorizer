import type { PartsStyleFunction } from "@chakra-ui/theme-tools";
declare const _default: {
    parts: ("root" | "field" | "stepperGroup" | "stepper")[];
    baseStyle: PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"root" | "field" | "stepperGroup" | "stepper">, "parts">>;
    sizes: {
        xs: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
        sm: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
        md: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
        lg: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
    };
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
//# sourceMappingURL=number-input.d.ts.map