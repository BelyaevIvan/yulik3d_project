import { request } from './client';
import type { FavoriteAddResponse, ItemCardDTO, ListPage } from './types';

export const favoritesApi = {
  list: (limit = 20, offset = 0) =>
    request<ListPage<ItemCardDTO>>('/favorites', { query: { limit, offset } }),
  add: (itemID: string) => request<FavoriteAddResponse>(`/favorites/${itemID}`, { method: 'POST' }),
  remove: (itemID: string) => request<void>(`/favorites/${itemID}`, { method: 'DELETE' }),
};
