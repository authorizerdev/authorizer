import { Target, TargetAndTransition, Transition } from "framer-motion";
declare type TargetResolver<P = {}> = (props: P & {
    transition?: TransitionConfig;
    transitionEnd?: TransitionEndConfig;
    delay?: number | DelayConfig;
}) => TargetAndTransition;
declare type Variant<P = {}> = TargetAndTransition | TargetResolver<P>;
export declare type Variants<P = {}> = {
    enter: Variant<P>;
    exit: Variant<P>;
    initial?: Variant<P>;
};
declare type WithMotionState<P> = Partial<Record<"enter" | "exit", P>>;
export declare type TransitionConfig = WithMotionState<Transition>;
export declare type TransitionEndConfig = WithMotionState<Target>;
export declare type DelayConfig = WithMotionState<number>;
export declare const TransitionEasings: {
    readonly ease: readonly [0.25, 0.1, 0.25, 1];
    readonly easeIn: readonly [0.4, 0, 1, 1];
    readonly easeOut: readonly [0, 0, 0.2, 1];
    readonly easeInOut: readonly [0.4, 0, 0.2, 1];
};
export declare const TransitionVariants: {
    scale: {
        enter: {
            scale: number;
        };
        exit: {
            scale: number;
        };
    };
    fade: {
        enter: {
            opacity: number;
        };
        exit: {
            opacity: number;
        };
    };
    pushLeft: {
        enter: {
            x: string;
        };
        exit: {
            x: string;
        };
    };
    pushRight: {
        enter: {
            x: string;
        };
        exit: {
            x: string;
        };
    };
    pushUp: {
        enter: {
            y: string;
        };
        exit: {
            y: string;
        };
    };
    pushDown: {
        enter: {
            y: string;
        };
        exit: {
            y: string;
        };
    };
    slideLeft: {
        position: {
            left: number;
            top: number;
            bottom: number;
            width: string;
        };
        enter: {
            x: number;
            y: number;
        };
        exit: {
            x: string;
            y: number;
        };
    };
    slideRight: {
        position: {
            right: number;
            top: number;
            bottom: number;
            width: string;
        };
        enter: {
            x: number;
            y: number;
        };
        exit: {
            x: string;
            y: number;
        };
    };
    slideUp: {
        position: {
            top: number;
            left: number;
            right: number;
            maxWidth: string;
        };
        enter: {
            x: number;
            y: number;
        };
        exit: {
            x: number;
            y: string;
        };
    };
    slideDown: {
        position: {
            bottom: number;
            left: number;
            right: number;
            maxWidth: string;
        };
        enter: {
            x: number;
            y: number;
        };
        exit: {
            x: number;
            y: string;
        };
    };
};
export declare type SlideDirection = "top" | "left" | "bottom" | "right";
export declare function slideTransition(options?: {
    direction?: SlideDirection;
}): {
    position: {
        left: number;
        top: number;
        bottom: number;
        width: string;
    };
    enter: {
        x: number;
        y: number;
    };
    exit: {
        x: string;
        y: number;
    };
} | {
    position: {
        right: number;
        top: number;
        bottom: number;
        width: string;
    };
    enter: {
        x: number;
        y: number;
    };
    exit: {
        x: string;
        y: number;
    };
} | {
    position: {
        top: number;
        left: number;
        right: number;
        maxWidth: string;
    };
    enter: {
        x: number;
        y: number;
    };
    exit: {
        x: number;
        y: string;
    };
} | {
    position: {
        bottom: number;
        left: number;
        right: number;
        maxWidth: string;
    };
    enter: {
        x: number;
        y: number;
    };
    exit: {
        x: number;
        y: string;
    };
};
export declare const TransitionDefaults: {
    readonly enter: {
        readonly duration: 0.2;
        readonly ease: readonly [0, 0, 0.2, 1];
    };
    readonly exit: {
        readonly duration: 0.1;
        readonly ease: readonly [0.4, 0, 1, 1];
    };
};
export declare type WithTransitionConfig<P extends object> = Omit<P, "transition"> & {
    /**
     * If `true`, the element will unmount when `in={false}` and animation is done
     */
    unmountOnExit?: boolean;
    /**
     * Show the component; triggers the enter or exit states
     */
    in?: boolean;
    /**
     * Custom `transition` definition for `enter` and `exit`
     */
    transition?: TransitionConfig;
    /**
     * Custom `transitionEnd` definition for `enter` and `exit`
     */
    transitionEnd?: TransitionEndConfig;
    /**
     * Custom `delay` definition for `enter` and `exit`
     */
    delay?: number | DelayConfig;
};
export declare const withDelay: {
    enter: (transition: Transition, delay?: number | Partial<Record<"exit" | "enter", number>> | undefined) => {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type?: "tween" | undefined;
        duration?: number | undefined;
        ease?: import("framer-motion/types/types").Easing | import("framer-motion/types/types").Easing[] | undefined;
        elapsed?: number | undefined;
        times?: number[] | undefined;
        easings?: import("framer-motion/types/types").Easing[] | undefined;
        from?: string | number | undefined;
        to?: import("framer-motion").ValueTarget | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "spring";
        stiffness?: number | undefined;
        damping?: number | undefined;
        mass?: number | undefined;
        duration?: number | undefined;
        bounce?: number | undefined;
        restSpeed?: number | undefined;
        restDelta?: number | undefined;
        from?: string | number | undefined;
        to?: import("framer-motion").ValueTarget | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "keyframes";
        values: import("framer-motion").KeyframesTarget;
        times?: number[] | undefined;
        ease?: import("framer-motion/types/types").Easing | import("framer-motion/types/types").Easing[] | undefined;
        easings?: import("framer-motion/types/types").Easing | import("framer-motion/types/types").Easing[] | undefined;
        elapsed?: number | undefined;
        duration?: number | undefined;
        from?: string | number | undefined;
        to?: import("framer-motion").ValueTarget | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "inertia";
        modifyTarget?(v: number): number;
        bounceStiffness?: number | undefined;
        bounceDamping?: number | undefined;
        power?: number | undefined;
        timeConstant?: number | undefined;
        restDelta?: number | undefined;
        min?: number | undefined;
        max?: number | undefined;
        from?: string | number | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "just";
        to?: import("framer-motion").ValueTarget | undefined;
        from?: string | number | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: false;
        from?: string | number | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
    };
    exit: (transition: Transition, delay?: number | Partial<Record<"exit" | "enter", number>> | undefined) => {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type?: "tween" | undefined;
        duration?: number | undefined;
        ease?: import("framer-motion/types/types").Easing | import("framer-motion/types/types").Easing[] | undefined;
        elapsed?: number | undefined;
        times?: number[] | undefined;
        easings?: import("framer-motion/types/types").Easing[] | undefined;
        from?: string | number | undefined;
        to?: import("framer-motion").ValueTarget | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "spring";
        stiffness?: number | undefined;
        damping?: number | undefined;
        mass?: number | undefined;
        duration?: number | undefined;
        bounce?: number | undefined;
        restSpeed?: number | undefined;
        restDelta?: number | undefined;
        from?: string | number | undefined;
        to?: import("framer-motion").ValueTarget | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "keyframes";
        values: import("framer-motion").KeyframesTarget;
        times?: number[] | undefined;
        ease?: import("framer-motion/types/types").Easing | import("framer-motion/types/types").Easing[] | undefined;
        easings?: import("framer-motion/types/types").Easing | import("framer-motion/types/types").Easing[] | undefined;
        elapsed?: number | undefined;
        duration?: number | undefined;
        from?: string | number | undefined;
        to?: import("framer-motion").ValueTarget | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "inertia";
        modifyTarget?(v: number): number;
        bounceStiffness?: number | undefined;
        bounceDamping?: number | undefined;
        power?: number | undefined;
        timeConstant?: number | undefined;
        restDelta?: number | undefined;
        min?: number | undefined;
        max?: number | undefined;
        from?: string | number | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: "just";
        to?: import("framer-motion").ValueTarget | undefined;
        from?: string | number | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
        type: false;
        from?: string | number | undefined;
        velocity?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
    } | {
        delay: number | undefined;
        when?: string | false | undefined;
        delayChildren?: number | undefined;
        staggerChildren?: number | undefined;
        staggerDirection?: number | undefined;
        repeat?: number | undefined;
        repeatType?: "reverse" | "loop" | "mirror" | undefined;
        repeatDelay?: number | undefined;
    };
};
export {};
//# sourceMappingURL=transition-utils.d.ts.map