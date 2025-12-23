/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_BASE_SHORT_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
