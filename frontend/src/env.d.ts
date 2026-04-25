/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_BACKEND_URL?: string;
  readonly VITE_CONTACT_EMAIL?: string;
  readonly VITE_CONTACT_VK?: string;
  readonly VITE_CONTACT_TG?: string;
  readonly VITE_CONTACT_INSTAGRAM?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
