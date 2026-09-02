import { useEffect } from "react";
import { Outlet, useMatches } from "react-router-dom";

type PageTitleHandle = { title?: string };

function PageTitle() {
    const matches = useMatches();

    useEffect(() => {
        const title = [...matches].reverse().map((match) => (match.handle as PageTitleHandle | undefined)?.title).find(Boolean);
        document.title = `${title || "Halaman Tidak Ditemukan"} | Devotion`;
    }, [matches]);

    return <Outlet />;
}

export default PageTitle;
