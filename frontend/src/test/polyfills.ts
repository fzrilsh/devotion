import { TextDecoder, TextEncoder } from "node:util";

// Beberapa dependency browser-facing memakai TextEncoder/TextDecoder yang tidak
// selalu ada di lingkungan jsdom. Kita pasang keduanya di globalThis supaya
// test dan runtime browser-like tetap punya API yang sama seperti di browser.
Object.assign(globalThis, { TextEncoder, TextDecoder });

// jsdom tidak mengimplementasikan Blob.prototype.arrayBuffer maupun
// URL.createObjectURL, keduanya dipakai pratinjau berkas. FileReader ada di
// jsdom, jadi dipakai sebagai jembatan untuk arrayBuffer.
if (typeof Blob !== "undefined" && !Blob.prototype.arrayBuffer) {
    Blob.prototype.arrayBuffer = function (): Promise<ArrayBuffer> {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve(reader.result as ArrayBuffer);
            reader.onerror = () => reject(reader.error);
            reader.readAsArrayBuffer(this);
        });
    };
}

if (typeof URL.createObjectURL === "undefined") {
    let objectUrlCounter = 0;

    URL.createObjectURL = () => `blob:mock-${++objectUrlCounter}`;
    URL.revokeObjectURL = () => {};
}
