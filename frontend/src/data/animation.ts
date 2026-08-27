import { easeInOut, easeOut } from "motion";

const smooth = {
    duration: 0.6,
    ease: easeInOut,
};

const fast = {
    duration: 0.4,
    ease: easeOut,
};

const slow = {
    duration: 0.8,
    ease: easeInOut,
};

export const fadeIn = {
    initial: {
        opacity: 0,
    },
    whileInView: {
        opacity: 1,
    },
    transition: smooth,
};

export const fadeScaleIn = {
    initial: {
        opacity: 0,
        scale: 0.8,
    },
    whileInView: {
        opacity: 1,
        scale: 1,
    },
    transition: smooth,
};

export const fadeScaleSmall = {
    initial: {
        opacity: 0,
        scale: 0.95,
    },
    whileInView: {
        opacity: 1,
        scale: 1,
    },
    transition: fast,
};

export const slideUp = {
    initial: {
        opacity: 0,
        y: 40,
    },
    whileInView: {
        opacity: 1,
        y: 0,
    },
    transition: smooth,
};

export const slideDown = {
    initial: {
        opacity: 0,
        y: -40,
    },
    whileInView: {
        opacity: 1,
        y: 0,
    },
    transition: smooth,
};

export const slideLeft = {
    initial: {
        opacity: 0,
        x: 40,
    },
    whileInView: {
        opacity: 1,
        x: 0,
    },
    transition: smooth,
};

export const slideRight = {
    initial: {
        opacity: 0,
        x: -40,
    },
    whileInView: {
        opacity: 1,
        x: 0,
    },
    transition: smooth,
};

export const zoomIn = {
    initial: {
        scale: 0.5,
        opacity: 0,
    },
    whileInView: {
        scale: 1,
        opacity: 1,
    },
    transition: slow,
};

export const zoomOut = {
    initial: {
        scale: 1.2,
        opacity: 0,
    },
    whileInView: {
        scale: 1,
        opacity: 1,
    },
    transition: slow,
};

export const popIn = {
    initial: {
        opacity: 0,
        scale: 0.7,
    },
    whileInView: {
        opacity: 1,
        scale: 1,
    },
    transition: {
        duration: 0.5,
        ease: easeOut,
    },
};

export const blurIn = {
    initial: {
        opacity: 0,
        filter: "blur(10px)",
    },
    whileInView: {
        opacity: 1,
        filter: "blur(0px)",
    },
    transition: slow,
};

export const blurSlideUp = {
    initial: {
        opacity: 0,
        y: 30,
        filter: "blur(8px)",
    },
    whileInView: {
        opacity: 1,
        y: 0,
        filter: "blur(0px)",
    },
    transition: smooth,
};

export const rotateIn = {
    initial: {
        opacity: 0,
        rotate: -10,
        scale: 0.9,
    },
    whileInView: {
        opacity: 1,
        rotate: 0,
        scale: 1,
    },
    transition: smooth,
};

export const flipIn = {
    initial: {
        opacity: 0,
        rotateY: 90,
    },
    whileInView: {
        opacity: 1,
        rotateY: 0,
    },
    transition: slow,
};

export const slideScaleIn = {
    initial: {
        opacity: 0,
        x: 50,
        scale: 0.9,
    },
    whileInView: {
        opacity: 1,
        x: 0,
        scale: 1,
    },
    transition: smooth,
};

export const dropIn = {
    initial: {
        opacity: 0,
        y: -80,
        scale: 0.9,
    },
    whileInView: {
        opacity: 1,
        y: 0,
        scale: 1,
    },
    transition: {
        duration: 0.6,
        ease: easeOut,
    },
};