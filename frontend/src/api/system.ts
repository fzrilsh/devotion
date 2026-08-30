import { apiClient } from "./client";
import type { components } from "./types";

export type Health = components["schemas"]["Health"];

// Endpoint publik (security: [] pada kontrak) dan dipakai juga oleh pemantau
// uptime. Basis data gagal atau penyimpanan penuh dibalas 503, jadi pemanggil
// harus memperlakukan galat sebagai keadaan yang tetap perlu ditampilkan.
export async function getHealth(): Promise<Health> {
    return apiClient<Health>("/health");
}
