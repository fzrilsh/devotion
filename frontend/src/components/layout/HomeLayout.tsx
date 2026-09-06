import type { ReactNode } from "react";
import Header from "./Header/Header";
import Footer from "./Footer/Footer";

type HomeLayoutProps = {
    children?: ReactNode;
};

export default function HomeLayout({ children }: HomeLayoutProps) {
    return (
        <>
            <Header />
            <main>{children}</main>
            <Footer />
        </>
    );
}
