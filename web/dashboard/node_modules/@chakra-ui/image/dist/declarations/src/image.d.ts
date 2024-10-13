import { SystemProps, HTMLChakraProps } from "@chakra-ui/system";
import * as React from "react";
import { UseImageProps } from "./use-image";
interface NativeImageOptions {
    /**
     * The native HTML `width` attribute to the passed to the `img`
     */
    htmlWidth?: string | number;
    /**
     * The native HTML `height` attribute to the passed to the `img`
     */
    htmlHeight?: string | number;
}
interface ImageOptions extends NativeImageOptions {
    /**
     * Fallback image `src` to show if image is loading or image fails.
     *
     * Note 🚨: We recommend you use a local image
     */
    fallbackSrc?: string;
    /**
     * Fallback element to show if image is loading or image fails.
     * @type React.ReactElement
     */
    fallback?: React.ReactElement;
    /**
     * Defines loading strategy
     */
    loading?: "eager" | "lazy";
    /**
     * How the image to fit within its bounds.
     * It maps to css `object-fit` property.
     * @type SystemProps["objectFit"]
     */
    fit?: SystemProps["objectFit"];
    /**
     * How to align the image within its bounds.
     * It maps to css `object-position` property.
     * @type SystemProps["objectPosition"]
     */
    align?: SystemProps["objectPosition"];
    /**
     * If `true`, opt out of the `fallbackSrc` logic and use as `img`
     */
    ignoreFallback?: boolean;
}
export interface ImageProps extends UseImageProps, Omit<HTMLChakraProps<"img">, keyof UseImageProps>, ImageOptions {
}
/**
 * React component that renders an image with support
 * for fallbacks
 *
 * @see Docs https://chakra-ui.com/image
 */
export declare const Image: import("@chakra-ui/system").ComponentWithAs<"img", ImageProps>;
export interface ImgProps extends HTMLChakraProps<"img">, NativeImageOptions {
}
/**
 * Fallback component for most SSR users who want to use the native `img` with
 * support for chakra props
 */
export declare const Img: import("@chakra-ui/system").ComponentWithAs<"img", ImgProps>;
export {};
//# sourceMappingURL=image.d.ts.map