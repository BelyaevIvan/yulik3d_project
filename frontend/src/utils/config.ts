// Контактные данные — единственное место, откуда тянутся ссылки.
const env = (import.meta as any).env || {};

export const config = {
  apiBaseUrl: '/api/v1',
  contact: {
    email: env.VITE_CONTACT_EMAIL || 'youbob9898@mail.ru',
    vk: env.VITE_CONTACT_VK || 'https://vk.ru/serlyberly',
    telegram: env.VITE_CONTACT_TG || 'https://t.me/vna2revova',
    instagram: env.VITE_CONTACT_INSTAGRAM || 'https://instagram.com',
  },
};
