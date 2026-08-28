import { TextDecoder, TextEncoder } from "node:util";

// jsdom tidak menyediakan TextEncoder/TextDecoder yang dipakai react-router.
Object.assign(globalThis, { TextEncoder, TextDecoder });
