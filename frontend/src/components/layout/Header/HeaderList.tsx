import React from "react";
import HeaderLink from "./HeaderLink";
import { useHeader } from "./useHeader";

interface NavListProps {
    items?: Array<{ label: string; href: string }>;
}

export default function HeaderList({ items = [] }: NavListProps) {
    const { activeLink, handleClick } = useHeader(items, 120);

    return (
        <>
            {items.map((item, index) => (
                <React.Fragment key={index}>
                    <HeaderLink href={item.href} active={item.href === activeLink} onClick={(e: React.MouseEvent<HTMLAnchorElement>) => handleClick(e, item.href)}>
                        {item.label}
                    </HeaderLink>
                    {index < items.length - 1 && (
                        <span className="mx-2 text-slate-500" aria-label="hidden">
                            /
                        </span>
                    )}
                </React.Fragment>
            ))}
        </>
    );
}
