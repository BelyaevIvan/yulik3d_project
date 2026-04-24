// DTO, отзеркаленные с бэкенда (см. backend/internal/model).
// При изменениях контракта — обновлять оба места.

export type Role = 'user' | 'admin';

export interface UserDTO {
  id: string;
  email: string;
  full_name: string;
  phone?: string | null;
  role: Role;
  created_at: string;
}

export interface ErrorResponse {
  statusCode: number;
  url: string;
  message: string;
  date: string;
}

export type CategoryType = 'figure' | 'other';

export interface CategoryDTO {
  id: string;
  name: string;
  type: CategoryType;
  subcategories?: SubcategoryShortDTO[];
}

export interface CategoryShortDTO {
  id: string;
  name: string;
  type: CategoryType;
}

export interface SubcategoryShortDTO {
  id: string;
  name: string;
}

export interface SubcategoryDTO {
  id: string;
  name: string;
  category_id: string;
  created_at: string;
}

export interface SubcategoryWithCategoryDTO {
  id: string;
  name: string;
  category: CategoryShortDTO;
}

export interface PictureDTO {
  id: string;
  url: string;
  position: number;
}

export interface OptionTypeShortDTO {
  id: string;
  code: string;
  label: string;
}

export interface OptionTypeDTO {
  id: string;
  code: string;
  label: string;
  created_at: string;
}

export interface ItemOptionValueDTO {
  id: string;
  value: string;
  price: number;
  position: number;
}

export interface OptionGroupDTO {
  type: OptionTypeShortDTO;
  values: ItemOptionValueDTO[];
}

export interface ItemCardDTO {
  id: string;
  name: string;
  articul: string;
  price: number;
  sale: number;
  final_price: number;
  hidden?: boolean;
  primary_picture_url: string | null;
  category: CategoryShortDTO | null;
  subcategories: SubcategoryShortDTO[];
}

export interface ItemDetailDTO {
  id: string;
  name: string;
  articul: string;
  description_info: string;
  description_other: string;
  price: number;
  sale: number;
  final_price: number;
  hidden: boolean;
  pictures: PictureDTO[];
  options: OptionGroupDTO[];
  subcategories: SubcategoryWithCategoryDTO[];
  created_at: string;
  updated_at: string;
}

export interface ItemOptionDTO {
  id: string;
  item_id: string;
  type: OptionTypeShortDTO;
  value: string;
  price: number;
  position: number;
}

export interface ListPage<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface FavoriteAddResponse {
  item_id: string;
  created_at: string;
}

export type OrderStatus =
  | 'created'
  | 'confirmed'
  | 'manufacturing'
  | 'delivering'
  | 'completed'
  | 'cancelled';

export interface OrderItemDTO {
  id: string;
  item_id: string;
  item_name_snapshot: string;
  item_articul_snapshot: string;
  quantity: number;
  unit_base_price: number;
  unit_total_price: number;
  options: OrderItemOptionDTO[];
}

export interface OrderItemOptionDTO {
  type_code_snapshot: string;
  type_label_snapshot: string;
  value_snapshot: string;
  price_snapshot: number;
}

export interface OrderListItemDTO {
  id: string;
  status: OrderStatus;
  total_price: number;
  items_count: number;
  created_at: string;
  updated_at: string;
}

export interface OrderDetailDTO {
  id: string;
  status: OrderStatus;
  total_price: number;
  customer_comment: string | null;
  contact_phone: string;
  contact_full_name: string;
  items: OrderItemDTO[];
  created_at: string;
  updated_at: string;
}

export interface OrderAdminListItemDTO {
  id: string;
  user: { id: string; email: string; full_name: string };
  status: OrderStatus;
  total_price: number;
  items_count: number;
  contact_phone: string;
  contact_full_name: string;
  customer_comment: string | null;
  admin_note: string | null;
  created_at: string;
  updated_at: string;
}

export interface OrderAdminDetailDTO extends OrderDetailDTO {
  user: { id: string; email: string; full_name: string; phone?: string | null };
  admin_note: string | null;
}

export interface OrderItemCreate {
  item_id: string;
  quantity: number;
  option_ids: string[];
}

export interface OrderCreateRequest {
  items: OrderItemCreate[];
  customer_comment?: string;
  contact_phone: string;
  contact_full_name: string;
}
