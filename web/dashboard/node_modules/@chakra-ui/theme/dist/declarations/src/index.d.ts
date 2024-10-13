import type { ThemeConfig } from "./theme.types";
export declare const theme: {
    components: {
        Accordion: {
            parts: ("button" | "icon" | "container" | "item" | "panel")[];
            baseStyle: Partial<Record<"button" | "icon" | "container" | "item" | "panel", import("@chakra-ui/styled-system").CSSObject>>;
        };
        Alert: {
            parts: ("title" | "description" | "icon" | "container")[];
            baseStyle: Partial<Record<"title" | "description" | "icon" | "container", import("@chakra-ui/styled-system").CSSObject>>;
            variants: {
                subtle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
                "left-accent": import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
                "top-accent": import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
                solid: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"title" | "description" | "icon" | "container">, "parts">>;
            };
            defaultProps: {
                variant: string;
                colorScheme: string;
            };
        };
        Avatar: {
            parts: ("label" | "group" | "container" | "badge" | "excessLabel")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"label" | "group" | "container" | "badge" | "excessLabel">, "parts">>;
            sizes: {
                "2xs": Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                xs: Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                sm: Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                md: Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                lg: Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                xl: Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                "2xl": Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
                full: Partial<Record<"label" | "group" | "container" | "badge" | "excessLabel", import("@chakra-ui/styled-system").CSSObject>>;
            };
            defaultProps: {
                size: string;
            };
        };
        Badge: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            variants: {
                solid: import("@chakra-ui/theme-tools").SystemStyleFunction;
                subtle: import("@chakra-ui/theme-tools").SystemStyleFunction;
                outline: import("@chakra-ui/theme-tools").SystemStyleFunction;
            };
            defaultProps: {
                variant: string;
                colorScheme: string;
            };
        };
        Breadcrumb: {
            parts: ("link" | "separator" | "container" | "item")[];
            baseStyle: Partial<Record<"link" | "separator" | "container" | "item", import("@chakra-ui/styled-system").CSSObject>>;
        };
        Button: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            variants: {
                ghost: import("@chakra-ui/theme-tools").SystemStyleFunction;
                outline: import("@chakra-ui/theme-tools").SystemStyleFunction;
                solid: import("@chakra-ui/theme-tools").SystemStyleFunction;
                link: import("@chakra-ui/theme-tools").SystemStyleFunction;
                unstyled: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            };
            sizes: Record<string, import("@chakra-ui/styled-system").CSSObject>;
            defaultProps: {
                variant: string;
                size: string;
                colorScheme: string;
            };
        };
        Checkbox: {
            parts: ("label" | "icon" | "container" | "control")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"label" | "icon" | "container" | "control">, "parts">>;
            sizes: Record<string, Partial<Record<"label" | "icon" | "container" | "control", import("@chakra-ui/styled-system").CSSObject>>>;
            defaultProps: {
                size: string;
                colorScheme: string;
            };
        };
        CloseButton: {
            baseStyle: import("@chakra-ui/theme-tools").SystemStyleFunction;
            sizes: Record<string, import("@chakra-ui/styled-system").CSSObject>;
            defaultProps: {
                size: string;
            };
        };
        Code: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            variants: {
                solid: import("@chakra-ui/theme-tools").SystemStyleFunction;
                subtle: import("@chakra-ui/theme-tools").SystemStyleFunction;
                outline: import("@chakra-ui/theme-tools").SystemStyleFunction;
            };
            defaultProps: {
                variant: string;
                colorScheme: string;
            };
        };
        Container: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
        };
        Divider: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            variants: {
                solid: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
                dashed: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            };
            defaultProps: {
                variant: string;
            };
        };
        Drawer: {
            parts: ("body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer">, "parts">>;
            sizes: {
                xs: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                sm: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                md: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                lg: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                xl: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                full: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
            };
            defaultProps: {
                size: string;
            };
        };
        Editable: {
            parts: ("input" | "preview")[];
            baseStyle: Partial<Record<"input" | "preview", import("@chakra-ui/styled-system").CSSObject>>;
        };
        Form: {
            parts: ("container" | "helperText" | "requiredIndicator")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"container" | "helperText" | "requiredIndicator">, "parts">>;
        };
        FormLabel: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
        };
        Heading: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            sizes: Record<string, import("@chakra-ui/styled-system").CSSObject>;
            defaultProps: {
                size: string;
            };
        };
        Input: {
            parts: ("element" | "field" | "addon")[];
            baseStyle: Partial<Record<"element" | "field" | "addon", import("@chakra-ui/styled-system").CSSObject>>;
            sizes: Record<string, Partial<Record<"element" | "field" | "addon", import("@chakra-ui/styled-system").CSSObject>>>;
            variants: {
                outline: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                filled: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                flushed: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                unstyled: Partial<Record<"element" | "field" | "addon", import("@chakra-ui/styled-system").CSSObject>>;
            };
            defaultProps: {
                size: string;
                variant: string;
            };
        };
        Kbd: {
            baseStyle: import("@chakra-ui/theme-tools").SystemStyleFunction;
        };
        Link: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
        };
        List: {
            parts: ("icon" | "container" | "item")[];
            baseStyle: Partial<Record<"icon" | "container" | "item", import("@chakra-ui/styled-system").CSSObject>>;
        };
        Menu: {
            parts: ("button" | "list" | "item" | "command" | "divider" | "groupTitle")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"button" | "list" | "item" | "command" | "divider" | "groupTitle">, "parts">>;
        };
        Modal: {
            parts: ("body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer">, "parts">>;
            sizes: {
                xs: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                sm: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                md: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                lg: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                xl: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                "2xl": Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                "3xl": Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                "4xl": Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                "5xl": Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                "6xl": Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
                full: Partial<Record<"body" | "dialog" | "footer" | "header" | "overlay" | "closeButton" | "dialogContainer", import("@chakra-ui/styled-system").CSSObject>>;
            };
            defaultProps: {
                size: string;
            };
        };
        NumberInput: {
            parts: ("root" | "field" | "stepperGroup" | "stepper")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"root" | "field" | "stepperGroup" | "stepper">, "parts">>;
            sizes: {
                xs: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
                sm: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
                md: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
                lg: Partial<Record<"root" | "field" | "stepperGroup" | "stepper", import("@chakra-ui/styled-system").CSSObject>>;
            };
            variants: {
                outline: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                filled: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                flushed: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                unstyled: Partial<Record<"element" | "field" | "addon", import("@chakra-ui/styled-system").CSSObject>>;
            };
            defaultProps: {
                size: string;
                variant: string;
            };
        };
        PinInput: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            sizes: Record<string, import("@chakra-ui/styled-system").CSSObject>;
            variants: Record<string, import("@chakra-ui/theme-tools").SystemStyleInterpolation>;
            defaultProps: {
                size: string;
                variant: string;
            };
        };
        Popover: {
            parts: ("body" | "footer" | "header" | "content" | "closeButton" | "popper" | "arrow")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"body" | "footer" | "header" | "content" | "closeButton" | "popper" | "arrow">, "parts">>;
        };
        Progress: {
            parts: ("label" | "track" | "filledTrack")[];
            sizes: Record<string, Partial<Record<"label" | "track" | "filledTrack", import("@chakra-ui/styled-system").CSSObject>>>;
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"label" | "track" | "filledTrack">, "parts">>;
            defaultProps: {
                size: string;
                colorScheme: string;
            };
        };
        Radio: {
            parts: ("label" | "container" | "control")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"label" | "container" | "control">, "parts">>;
            sizes: Record<string, Partial<Record<"label" | "container" | "control", import("@chakra-ui/styled-system").CSSObject>>>;
            defaultProps: {
                size: string;
                colorScheme: string;
            };
        };
        Select: {
            parts: ("icon" | "field")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"icon" | "field">, "parts">>;
            sizes: Record<string, Partial<Record<"icon" | "field", import("@chakra-ui/styled-system").CSSObject>>>;
            variants: {
                outline: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                filled: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                flushed: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"element" | "field" | "addon">, "parts">>;
                unstyled: Partial<Record<"element" | "field" | "addon", import("@chakra-ui/styled-system").CSSObject>>;
            };
            defaultProps: {
                size: string;
                variant: string;
            };
        };
        Skeleton: {
            baseStyle: import("@chakra-ui/theme-tools").SystemStyleFunction;
        };
        SkipLink: {
            baseStyle: import("@chakra-ui/theme-tools").SystemStyleFunction;
        };
        Slider: {
            parts: ("track" | "container" | "thumb" | "filledTrack")[];
            sizes: {
                lg: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"track" | "container" | "thumb" | "filledTrack">, "parts">>;
                md: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"track" | "container" | "thumb" | "filledTrack">, "parts">>;
                sm: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"track" | "container" | "thumb" | "filledTrack">, "parts">>;
            };
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"track" | "container" | "thumb" | "filledTrack">, "parts">>;
            defaultProps: {
                size: string;
                colorScheme: string;
            };
        };
        Spinner: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            sizes: Record<string, import("@chakra-ui/styled-system").CSSObject>;
            defaultProps: {
                size: string;
            };
        };
        Stat: {
            parts: ("number" | "label" | "icon" | "container" | "helpText")[];
            baseStyle: Partial<Record<"number" | "label" | "icon" | "container" | "helpText", import("@chakra-ui/styled-system").CSSObject>>;
            sizes: Record<string, Partial<Record<"number" | "label" | "icon" | "container" | "helpText", import("@chakra-ui/styled-system").CSSObject>>>;
            defaultProps: {
                size: string;
            };
        };
        Switch: {
            parts: ("track" | "container" | "thumb")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"track" | "container" | "thumb">, "parts">>;
            sizes: Record<string, Partial<Record<"track" | "container" | "thumb", import("@chakra-ui/styled-system").CSSObject>>>;
            defaultProps: {
                size: string;
                colorScheme: string;
            };
        };
        Table: {
            parts: ("caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr")[];
            baseStyle: Partial<Record<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr", import("@chakra-ui/styled-system").CSSObject>>;
            variants: {
                simple: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr">, "parts">>;
                striped: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr">, "parts">>;
                unstyled: {};
            };
            sizes: Record<string, Partial<Record<"caption" | "table" | "tbody" | "td" | "tfoot" | "th" | "thead" | "tr", import("@chakra-ui/styled-system").CSSObject>>>;
            defaultProps: {
                variant: string;
                size: string;
                colorScheme: string;
            };
        };
        Tabs: {
            parts: ("tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator">, "parts">>;
            sizes: Record<string, Partial<Record<"tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator", import("@chakra-ui/styled-system").CSSObject>>>;
            variants: Record<string, import("@chakra-ui/theme-tools").PartsStyleInterpolation<Omit<import("@chakra-ui/theme-tools").Anatomy<"tab" | "tablist" | "tabpanel" | "tabpanels" | "root" | "indicator">, "parts">>>;
            defaultProps: {
                size: string;
                variant: string;
                colorScheme: string;
            };
        };
        Tag: {
            parts: ("label" | "container" | "closeButton")[];
            variants: Record<string, import("@chakra-ui/theme-tools").PartsStyleInterpolation<Omit<import("@chakra-ui/theme-tools").Anatomy<"label" | "container" | "closeButton">, "parts">>>;
            baseStyle: Partial<Record<"label" | "container" | "closeButton", import("@chakra-ui/styled-system").CSSObject>>;
            sizes: Record<string, Partial<Record<"label" | "container" | "closeButton", import("@chakra-ui/styled-system").CSSObject>>>;
            defaultProps: {
                size: string;
                variant: string;
                colorScheme: string;
            };
        };
        Textarea: {
            baseStyle: import("@chakra-ui/styled-system").CSSWithMultiValues | (import("@chakra-ui/styled-system").CSSWithMultiValues & import("@chakra-ui/styled-system").RecursivePseudo<import("@chakra-ui/styled-system").CSSWithMultiValues>);
            sizes: Record<string, import("@chakra-ui/styled-system").CSSObject>;
            variants: Record<string, import("@chakra-ui/theme-tools").SystemStyleInterpolation>;
            defaultProps: {
                size: string;
                variant: string;
            };
        };
        Tooltip: {
            baseStyle: import("@chakra-ui/theme-tools").SystemStyleFunction;
        };
        FormError: {
            parts: ("text" | "icon")[];
            baseStyle: import("@chakra-ui/theme-tools").PartsStyleFunction<Omit<import("@chakra-ui/theme-tools").Anatomy<"text" | "icon">, "parts">>;
        };
    };
    styles: import("@chakra-ui/theme-tools").Styles;
    config: ThemeConfig;
    /**
     * @deprecated
     * Duplicate theme type. Please use `Theme`
     */
    sizes: {
        container: {
            sm: string;
            md: string;
            lg: string;
            xl: string;
        };
        max: string;
        min: string;
        full: string;
        "3xs": string;
        "2xs": string;
        xs: string;
        sm: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
        "3xl": string;
        "4xl": string;
        "5xl": string;
        "6xl": string;
        "7xl": string;
        "8xl": string;
        px: string;
        0.5: string;
        1: string;
        1.5: string;
        2: string;
        2.5: string;
        3: string;
        3.5: string;
        4: string;
        5: string;
        6: string;
        7: string;
        8: string;
        9: string;
        10: string;
        12: string;
        14: string;
        16: string;
        20: string;
        24: string;
        28: string;
        32: string;
        36: string;
        40: string;
        44: string;
        48: string;
        52: string;
        56: string;
        60: string;
        64: string;
        72: string;
        80: string;
        96: string;
    };
    shadows: {
        xs: string;
        sm: string;
        base: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
        outline: string;
        inner: string;
        none: string;
        "dark-lg": string;
    };
    space: {
        px: string;
        0.5: string;
        1: string;
        1.5: string;
        2: string;
        2.5: string;
        3: string;
        3.5: string;
        4: string;
        5: string;
        6: string;
        7: string;
        8: string;
        9: string;
        10: string;
        12: string;
        14: string;
        16: string;
        20: string;
        24: string;
        28: string;
        32: string;
        36: string;
        40: string;
        44: string;
        48: string;
        52: string;
        56: string;
        60: string;
        64: string;
        72: string;
        80: string;
        96: string;
    };
    borders: {
        none: number;
        "1px": string;
        "2px": string;
        "4px": string;
        "8px": string;
    };
    transition: {
        property: {
            common: string;
            colors: string;
            dimensions: string;
            position: string;
            background: string;
        };
        easing: {
            "ease-in": string;
            "ease-out": string;
            "ease-in-out": string;
        };
        duration: {
            "ultra-fast": string;
            faster: string;
            fast: string;
            normal: string;
            slow: string;
            slower: string;
            "ultra-slow": string;
        };
    };
    letterSpacings: {
        tighter: string;
        tight: string;
        normal: string;
        wide: string;
        wider: string;
        widest: string;
    };
    lineHeights: {
        normal: string;
        none: number;
        shorter: number;
        short: number;
        base: number;
        tall: number;
        taller: string;
        "3": string;
        "4": string;
        "5": string;
        "6": string;
        "7": string;
        "8": string;
        "9": string;
        "10": string;
    };
    fontWeights: {
        hairline: number;
        thin: number;
        light: number;
        normal: number;
        medium: number;
        semibold: number;
        bold: number;
        extrabold: number;
        black: number;
    };
    fonts: {
        heading: string;
        body: string;
        mono: string;
    };
    fontSizes: {
        xs: string;
        sm: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
        "3xl": string;
        "4xl": string;
        "5xl": string;
        "6xl": string;
        "7xl": string;
        "8xl": string;
        "9xl": string;
    };
    breakpoints: import("@chakra-ui/theme-tools").Breakpoints<{
        sm: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
    }>;
    zIndices: {
        hide: number;
        auto: string;
        base: number;
        docked: number;
        dropdown: number;
        sticky: number;
        banner: number;
        overlay: number;
        modal: number;
        popover: number;
        skipLink: number;
        toast: number;
        tooltip: number;
    };
    radii: {
        none: string;
        sm: string;
        base: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
        "3xl": string;
        full: string;
    };
    blur: {
        none: number;
        sm: string;
        base: string;
        md: string;
        lg: string;
        xl: string;
        "2xl": string;
        "3xl": string;
    };
    colors: {
        transparent: string;
        current: string;
        black: string;
        white: string;
        whiteAlpha: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        blackAlpha: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        gray: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        red: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        orange: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        yellow: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        green: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        teal: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        blue: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        cyan: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        purple: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        pink: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        linkedin: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        facebook: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        messenger: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        whatsapp: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        twitter: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
        telegram: {
            50: string;
            100: string;
            200: string;
            300: string;
            400: string;
            500: string;
            600: string;
            700: string;
            800: string;
            900: string;
        };
    };
    direction: "ltr";
};
export declare type Theme = typeof theme;
/**
 * @deprecated
 * Duplicate theme type. Please use `Theme`
 */
export declare type DefaultChakraTheme = Theme;
export * from "./theme.types";
export * from "./utils";
export default theme;
//# sourceMappingURL=index.d.ts.map