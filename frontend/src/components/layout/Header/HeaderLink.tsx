import { cn } from "@lib/utils";
import type { MouseEventHandler, ReactNode } from "react";

interface NavLinkProps {
    href: string;
    children: ReactNode;
    liClass?: string;
    aClass?: string;
    active?: boolean;
    onClick?: MouseEventHandler<Element>;
}

export default function HeaderLink({ href, children, liClass, aClass, active = false, onClick }: NavLinkProps) {
    return (
        <li className={cn("flex-center w-auto", liClass)}>
            <a onClick={onClick} href={href} className={cn("px-3 py-2 font-medium transition-all duration-200 whitespace-nowrap border-b-4", aClass, active ? "font-semibold border-deep-navy-500 text-deep-navy-500" : "border-transparent text-slate-500")}>
                {children}
            </a>
        </li>
    );
}
