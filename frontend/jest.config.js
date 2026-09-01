/** @type {import('jest').Config} */
export default {
    preset: "ts-jest",
    testEnvironment: "jsdom",
    roots: ["<rootDir>/src"],
    setupFiles: ["<rootDir>/src/test/polyfills.ts"],
    setupFilesAfterEnv: ["<rootDir>/src/test/setup.ts"],
    moduleNameMapper: {
        "^@api/(.*)$": "<rootDir>/src/api/$1",
        "^@assets/(.*)$": "<rootDir>/src/assets/$1",
        "^@components/(.*)$": "<rootDir>/src/components/$1",
        "^@data/(.*)$": "<rootDir>/src/data/$1",
        "^@hooks/(.*)$": "<rootDir>/src/hooks/$1",
        "^@lib/(.*)$": "<rootDir>/src/lib/$1",
        "^@pages/(.*)$": "<rootDir>/src/pages/$1",
        "^@providers/(.*)$": "<rootDir>/src/providers/$1",
        "^@routes/(.*)$": "<rootDir>/src/routes/$1",
        "^@schemas/(.*)$": "<rootDir>/src/schemas/$1",
        "^@styles/(.*)$": "<rootDir>/src/styles/$1",
        "\\.(css|less|scss|sass)$": "identity-obj-proxy",
        "\\.(png|jpe?g|gif|svg|webp)$": "<rootDir>/src/test/fileMock.ts",
    },
    transform: {
        "^.+\\.tsx?$": [
            "ts-jest",
            {
                tsconfig: {
                    jsx: "react-jsx",
                    // Sejajar dengan tsconfig.app.json. Tanpa ini ts-jest memakai target
                    // bawaannya yang lebih tua, jadi metode seperti Array.prototype.at
                    // gagal typecheck di pengujian meski lolos di build aplikasi.
                    target: "es2023",
                    // ts-jest mengetik tiap berkas uji sendiri-sendiri, jadi import
                    // jest-dom di src/test/setup.ts tidak ikut terbaca. Tanpa entri
                    // ini, matcher seperti toBeInTheDocument ditolak TS2339.
                    types: ["jest", "node", "@testing-library/jest-dom"],
                    esModuleInterop: true,
                    allowImportingTsExtensions: true,
                    verbatimModuleSyntax: false,
                    noEmit: true,
                    // jest-dom v7 meletakkan tipe di balik kondisi exports;
                    // "bundler" yang membacanya, "node" (bawaan) tidak.
                    moduleResolution: "bundler",
                    baseUrl: ".",
                    paths: {
                        "@api/*": ["./src/api/*"],
                        "@assets/*": ["./src/assets/*"],
                        "@components/*": ["./src/components/*"],
                        "@data/*": ["./src/data/*"],
                        "@hooks/*": ["./src/hooks/*"],
                        "@lib/*": ["./src/lib/*"],
                        "@pages/*": ["./src/pages/*"],
                        "@providers/*": ["./src/providers/*"],
                        "@routes/*": ["./src/routes/*"],
                        "@schemas/*": ["./src/schemas/*"],
                        "@styles/*": ["./src/styles/*"],
                    },
                },
                diagnostics: {
                    ignoreCodes: ["TS5023"],
                },
            },
        ],
    },
    testMatch: ["**/?(*.)+(test).[jt]s?(x)"],
};
