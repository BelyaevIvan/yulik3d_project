import { request } from './client';
import type {
  CategoryDTO, CategoryType,
  ItemCardDTO, ItemDetailDTO, ListPage, SubcategoryShortDTO,
} from './types';

export interface CatalogQuery {
  category_type?: CategoryType;
  category_id?: string;
  subcategory_id?: string;
  q?: string;
  has_sale?: boolean;
  sort?: string;
  limit?: number;
  offset?: number;
}

export const catalogApi = {
  listItems: (q: CatalogQuery = {}) =>
    request<ListPage<ItemCardDTO>>('/items', { query: q as any }),

  getItem: (id: string) => request<ItemDetailDTO>(`/items/${id}`),

  listCategories: (type?: CategoryType, withSubcategories = false) =>
    request<{ categories: CategoryDTO[] }>('/categories', {
      query: { type, with_subcategories: withSubcategories || undefined },
    }),

  listSubcategories: (categoryID: string) =>
    request<{ subcategories: SubcategoryShortDTO[] }>(`/categories/${categoryID}/subcategories`),
};
