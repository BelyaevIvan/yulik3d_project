import { request } from './client';
import type {
  ListPage, OrderCreateRequest, OrderDetailDTO, OrderListItemDTO, OrderStatus,
} from './types';

export const ordersApi = {
  create: (req: OrderCreateRequest) =>
    request<OrderDetailDTO>('/orders', { method: 'POST', body: req }),

  listMy: (q: { status?: OrderStatus; limit?: number; offset?: number } = {}) =>
    request<ListPage<OrderListItemDTO>>('/orders', { query: q as any }),

  getMy: (id: string) => request<OrderDetailDTO>(`/orders/${id}`),
};
