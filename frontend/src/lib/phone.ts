/**
 * Converts a phone number typed by a person into the E.164 form the backend
 * accepts. `internal/account/handlers.go` matches the raw request body against
 * `^\+62[0-9]{8,13}$`, so the leading plus is mandatory on the wire even though
 * the column stores it without one.
 */
export function normalizePhone(value: string): string {
    const digits = value.replace(/\D/g, "");

    if (!digits) return "";

    if (digits.startsWith("0")) {
        return `+62${digits.slice(1)}`;
    }

    if (digits.startsWith("62")) {
        return `+${digits}`;
    }

    return `+62${digits}`;
}
