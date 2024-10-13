declare const _default: {
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
export default _default;
//# sourceMappingURL=index.d.ts.map