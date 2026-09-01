import { apiClient } from "./client";
import type { components } from "./types";

export type Health = components["schemas"]["Health"];

export async function getHealth(): Promise<Health> {
    return apiClient<Health>("/health");
}
