import { request } from './client';
import type {
  CategoryDTO, CategoryType, ItemCardDTO, ItemDetailDTO, ItemOptionDTO, ListPage,
  OptionTypeDTO, OrderAdminDetailDTO, OrderAdminListItemDTO, OrderStatus, PictureDTO,
  SubcategoryDTO,
} from './types';

export interface AdminItemListQuery {
  hidden?: 'any' | 'true' | 'false';
  category_type?: CategoryType;
  category_id?: string;
  subcategory_id?: string;
  q?: string;
  has_sale?: boolean;
  sort?: string;
  limit?: number;
  offset?: number;
}

export interface ItemOptionInput {
  type_id: string;
  value: string;
  price: number;
  position: number;
}

export interface ItemCreateRequest {
  name: string;
  description_info: string;
  description_other: string;
  price: number;
  sale: number;
  hidden: boolean;
  subcategory_ids: string[];
  options: ItemOptionInput[];
}

export const adminApi = {
  // Items
  listItems: (q: AdminItemListQuery = {}) =>
    request<ListPage<ItemCardDTO>>('/admin/items', { query: q as any }),
  createItem: (req: ItemCreateRequest) =>
    request<ItemDetailDTO>('/admin/items', { method: 'POST', body: req }),
  getItem: (id: string) => request<ItemDetailDTO>(`/admin/items/${id}`),
  updateItem: (id: string, req: ItemCreateRequest) =>
    request<ItemDetailDTO>(`/admin/items/${id}`, { method: 'PUT', body: req }),
  patchItem: (id: string, p: Partial<{ name: string; description_info: string; description_other: string; price: number; sale: number; hidden: boolean }>) =>
    request<ItemDetailDTO>(`/admin/items/${id}`, { method: 'PATCH', body: p }),

  // Pictures
  uploadPicture: (itemID: string, file: File, position?: number) => {
    const fd = new FormData();
    fd.append('file', file);
    if (position !== undefined) fd.append('position', String(position));
    return request<PictureDTO>(`/admin/items/${itemID}/pictures`, { method: 'POST', formData: fd });
  },
  deletePicture: (itemID: string, pictureID: string) =>
    request<void>(`/admin/items/${itemID}/pictures/${pictureID}`, { method: 'DELETE' }),
  reorderPictures: (itemID: string, order: { picture_id: string; position: number }[]) =>
    request<{ pictures: PictureDTO[] }>(`/admin/items/${itemID}/pictures/reorder`, {
      method: 'PATCH', body: { order },
    }),

  // Option types
  listOptionTypes: () => request<{ option_types: OptionTypeDTO[] }>('/admin/option-types'),
  createOptionType: (code: string, label: string) =>
    request<OptionTypeDTO>('/admin/option-types', { method: 'POST', body: { code, label } }),
  patchOptionType: (id: string, label: string) =>
    request<OptionTypeDTO>(`/admin/option-types/${id}`, { method: 'PATCH', body: { label } }),
  deleteOptionType: (id: string) =>
    request<void>(`/admin/option-types/${id}`, { method: 'DELETE' }),

  // Item options (отдельные)
  addItemOption: (itemID: string, opt: ItemOptionInput) =>
    request<ItemOptionDTO>(`/admin/items/${itemID}/options`, { method: 'POST', body: opt }),
  patchItemOption: (id: string, p: Partial<{ value: string; price: number; position: number }>) =>
    request<ItemOptionDTO>(`/admin/item-options/${id}`, { method: 'PATCH', body: p }),
  deleteItemOption: (id: string) =>
    request<void>(`/admin/item-options/${id}`, { method: 'DELETE' }),

  // Categories
  createCategory: (name: string, type: CategoryType) =>
    request<CategoryDTO>('/admin/categories', { method: 'POST', body: { name, type } }),
  patchCategory: (id: string, p: { name?: string; type?: CategoryType }) =>
    request<CategoryDTO>(`/admin/categories/${id}`, { method: 'PATCH', body: p }),
  deleteCategory: (id: string) =>
    request<void>(`/admin/categories/${id}`, { method: 'DELETE' }),
  createSubcategory: (categoryID: string, name: string) =>
    request<SubcategoryDTO>(`/admin/categories/${categoryID}/subcategories`, {
      method: 'POST', body: { name },
    }),
  patchSubcategory: (id: string, p: { name?: string; category_id?: string }) =>
    request<SubcategoryDTO>(`/admin/subcategories/${id}`, { method: 'PATCH', body: p }),
  deleteSubcategory: (id: string) =>
    request<void>(`/admin/subcategories/${id}`, { method: 'DELETE' }),

  // Orders
  listOrders: (q: { status?: OrderStatus; user_id?: string; q?: string; limit?: number; offset?: number } = {}) =>
    request<ListPage<OrderAdminListItemDTO>>('/admin/orders', { query: q as any }),
  getOrder: (id: string) => request<OrderAdminDetailDTO>(`/admin/orders/${id}`),
  patchOrderStatus: (id: string, status: OrderStatus) =>
    request<OrderAdminDetailDTO>(`/admin/orders/${id}/status`, { method: 'PATCH', body: { status } }),
  patchOrderNote: (id: string, admin_note: string | null) =>
    request<OrderAdminDetailDTO>(`/admin/orders/${id}`, { method: 'PATCH', body: { admin_note } }),

  // Main page (закрепления товаров на главной)
  mainPageList: () =>
    request<{ figures: ItemCardDTO[]; others: ItemCardDTO[] }>('/admin/main'),
  mainPagePin: (item_id: string, type: CategoryType, position?: number) =>
    request<{ ok: boolean }>('/admin/main', {
      method: 'POST',
      body: position !== undefined ? { item_id, type, position } : { item_id, type },
    }),
  mainPageUnpin: (type: CategoryType, item_id: string) =>
    request<{ ok: boolean }>(`/admin/main/${type}/${item_id}`, { method: 'DELETE' }),
  mainPageReorder: (type: CategoryType, order: { item_id: string; position: number }[]) =>
    request<{ ok: boolean }>(`/admin/main/${type}/reorder`, { method: 'PATCH', body: { order } }),
};
