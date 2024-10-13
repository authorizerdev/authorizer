export declare function getTransformTemplate(): string;
export declare function getTransformGpuTemplate(): string;
export declare const filterTemplate: {
    "--chakra-blur": string;
    "--chakra-brightness": string;
    "--chakra-contrast": string;
    "--chakra-grayscale": string;
    "--chakra-hue-rotate": string;
    "--chakra-invert": string;
    "--chakra-saturate": string;
    "--chakra-sepia": string;
    "--chakra-drop-shadow": string;
    filter: string;
};
export declare const backdropFilterTemplate: {
    backdropFilter: string;
    "--chakra-backdrop-blur": string;
    "--chakra-backdrop-brightness": string;
    "--chakra-backdrop-contrast": string;
    "--chakra-backdrop-grayscale": string;
    "--chakra-backdrop-hue-rotate": string;
    "--chakra-backdrop-invert": string;
    "--chakra-backdrop-opacity": string;
    "--chakra-backdrop-saturate": string;
    "--chakra-backdrop-sepia": string;
};
export declare function getRingTemplate(value: any): {
    "--chakra-ring-offset-shadow": string;
    "--chakra-ring-shadow": string;
    "--chakra-ring-width": any;
    boxShadow: string;
};
export declare const flexDirectionTemplate: {
    "row-reverse": {
        space: string;
        divide: string;
    };
    "column-reverse": {
        space: string;
        divide: string;
    };
};
export declare const spaceXTemplate: {
    "& > :not(style) ~ :not(style)": {
        marginInlineStart: string;
        marginInlineEnd: string;
    };
};
export declare const spaceYTemplate: {
    "& > :not(style) ~ :not(style)": {
        marginTop: string;
        marginBottom: string;
    };
};
//# sourceMappingURL=templates.d.ts.map