import { useAuth } from "@hooks/useAuth";

export function useAccountVerification() {
    const { user } = useAuth();

    const needsEmail = Boolean(user) && !user?.email_verified;
    const needsPhone = Boolean(user) && !user?.phone_verified;

    return {
        needsEmail,
        needsPhone,
        needsVerification: needsEmail || needsPhone,
    };
}
