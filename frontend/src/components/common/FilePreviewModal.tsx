import { apiUrl } from "@api/client";
import { cn } from "@lib/utils";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { motion } from "motion/react";
import { LuLoaderCircle, LuX } from "react-icons/lu";

const ZOOM_LEVELS = [1, 2, 4] as const;

type DetectedKind = "jpeg" | "png" | "pdf" | "unknown";

// Tipe ditebak dari magic bytes, bukan dari header yang dikirim server maupun
// nama berkas. Kontrak /files/{id} menyajikan application/octet-stream, jadi
// ini satu-satunya cara menentukan cara merender.
function detectKind(bytes: Uint8Array): DetectedKind {
    if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) return "jpeg";
    if (bytes.length >= 4 && bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47) return "png";
    if (bytes.length >= 4 && bytes[0] === 0x25 && bytes[1] === 0x50 && bytes[2] === 0x44 && bytes[3] === 0x46) return "pdf";
    return "unknown";
}

const MIME_BY_KIND: Record<Exclude<DetectedKind, "unknown">, string> = {
    jpeg: "image/jpeg",
    png: "image/png",
    pdf: "application/pdf",
};

// Konten pratinjau untuk satu berkas. Dirender dengan key berupa fileId, jadi
// pergantian berkas selalu remount dan state tidak perlu direset manual.
function FilePreviewContent({ fileId, title }: { fileId: string; title: string }) {
    const [objectUrl, setObjectUrl] = useState<string | null>(null);
    const [kind, setKind] = useState<DetectedKind>("unknown");
    const [error, setError] = useState(false);
    const [zoomIndex, setZoomIndex] = useState(0);

    useEffect(() => {
        let cancelled = false;
        let createdUrl: string | null = null;

        async function load() {
            try {
                const response = await fetch(apiUrl(`/files/${fileId}`), { credentials: "include" });
                if (!response.ok) throw new Error(`status ${response.status}`);

                const blob = await response.blob();
                const bytes = new Uint8Array(await blob.slice(0, 8).arrayBuffer());
                const detected = detectKind(bytes);
                if (cancelled) return;

                if (detected === "unknown") throw new Error("tipe berkas tidak dikenal");

                const typed = new Blob([blob], { type: MIME_BY_KIND[detected] });
                createdUrl = URL.createObjectURL(typed);
                setKind(detected);
                setObjectUrl(createdUrl);
            } catch {
                if (!cancelled) setError(true);
            }
        }

        load();

        return () => {
            cancelled = true;
            if (createdUrl) URL.revokeObjectURL(createdUrl);
        };
    }, [fileId]);

    function cycleZoom() {
        setZoomIndex((current) => (current + 1) % ZOOM_LEVELS.length);
    }

    const scale = ZOOM_LEVELS[zoomIndex];
    const isImage = kind === "jpeg" || kind === "png";

    if (error) {
        return (
            <div className="flex h-full items-center justify-center p-6">
                <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700" role="alert">
                    Berkas tidak dapat dimuat. Sesi Anda mungkin sudah berakhir. Silakan muat ulang halaman.
                </p>
            </div>
        );
    }

    if (objectUrl === null) {
        return (
            <div className="flex h-full items-center justify-center text-slate-400" aria-live="polite">
                <LuLoaderCircle className="size-8 animate-spin" aria-hidden />
                <span className="sr-only">Memuat pratinjau</span>
            </div>
        );
    }

    if (isImage) {
        return (
            <div className="flex h-full max-h-[70vh] items-center justify-center overflow-auto p-4">
                <img
                    src={objectUrl}
                    alt={title}
                    onClick={cycleZoom}
                    style={{ transform: `scale(${scale})` }}
                    className={cn("max-h-full max-w-full cursor-zoom-in object-contain transition-transform", scale > 1 && "cursor-move")}
                />
            </div>
        );
    }

    return <iframe src={objectUrl} title={title} className="h-[70vh] w-full" />;
}

function FilePreviewModal({ fileId, title, onClose }: { fileId: string | null; title: string; onClose: () => void }) {
    const isOpen = fileId !== null;

    useEffect(() => {
        if (!isOpen) return;

        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") onClose();
        }

        document.addEventListener("keydown", handleKeyDown);
        return () => document.removeEventListener("keydown", handleKeyDown);
    }, [isOpen, onClose]);

    if (!isOpen) return null;

    return createPortal(
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
            data-testid="backdrop"
            onClick={onClose}
            className="fixed inset-0 z-50 flex items-center justify-center bg-neutral-900/70 p-4 backdrop-blur-sm sm:p-8"
        >
            <div
                role="dialog"
                aria-modal="true"
                aria-label={`Pratinjau ${title}`}
                onClick={(event) => event.stopPropagation()}
                className="flex max-h-full w-full max-w-3xl flex-col overflow-hidden rounded-2xl bg-white shadow-xl"
            >
                <div className="flex items-center justify-between gap-3 border-b border-slate-200 px-5 py-3">
                    <p className="truncate text-sm font-bold text-slate-900">{title}</p>
                    <button
                        type="button"
                        onClick={onClose}
                        aria-label="Tutup"
                        autoFocus
                        className="cursor-pointer rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-100 hover:text-slate-700"
                    >
                        <LuX className="size-5" aria-hidden />
                    </button>
                </div>

                <div className="min-h-0 flex-1 overflow-hidden bg-slate-100">{fileId !== null ? <FilePreviewContent key={fileId} fileId={fileId} title={title} /> : null}</div>
            </div>
        </motion.div>,
        document.body,
    );
}

export default FilePreviewModal;
