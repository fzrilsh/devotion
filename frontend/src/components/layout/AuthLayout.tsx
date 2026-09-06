import type { ReactNode } from "react";

type AuthLayoutProps = {
    children?: ReactNode;
};

export default function AuthLayout({ children }: AuthLayoutProps) {
    return <main className="min-h-screen lg:grid lg:grid-cols-2">{children}</main>;
}
