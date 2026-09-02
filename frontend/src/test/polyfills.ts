import { TextDecoder, TextEncoder } from "node:util";

// Beberapa dependency browser-facing memakai TextEncoder/TextDecoder yang tidak
// selalu ada di lingkungan jsdom. Kita pasang keduanya di globalThis supaya
// test dan runtime browser-like tetap punya API yang sama seperti di browser.
Object.assign(globalThis, { TextEncoder, TextDecoder });
