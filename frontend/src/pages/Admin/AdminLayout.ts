// Общая обёртка админки — sidebar + main.
import { renderTemplate } from '@/utils/template';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import './AdminLayout.scss';

export type AdminSection = 'items' | 'categories' | 'options' | 'orders';

const tpl = `
<div class="admin">
  <aside class="admin__sidebar">
    <a href="/admin" data-link class="admin__sidebar-link {{#if (eq active "items")}}admin__sidebar-link--active{{/if}}">📦 Товары</a>
    <a href="/admin/orders" data-link class="admin__sidebar-link {{#if (eq active "orders")}}admin__sidebar-link--active{{/if}}">🧾 Заказы</a>
    <a href="/admin/categories" data-link class="admin__sidebar-link {{#if (eq active "categories")}}admin__sidebar-link--active{{/if}}">🏷️ Категории</a>
    <a href="/admin/option-types" data-link class="admin__sidebar-link {{#if (eq active "options")}}admin__sidebar-link--active{{/if}}">⚙️ Типы опций</a>
  </aside>
  <main class="admin__main" id="adminMain"></main>
</div>
`;

/** Гарантирует, что юзер — admin. Возвращает true если ок. */
export function requireAdmin(): boolean {
  if (!authStore.isAuthed()) {
    router.navigate('/login?next=' + encodeURIComponent(router.currentPath()));
    return false;
  }
  if (!authStore.isAdmin()) {
    router.replace('/403');
    return false;
  }
  return true;
}

/** Рендерит каркас админки и возвращает контейнер для контента. */
export function renderAdminShell(root: HTMLElement, active: AdminSection): HTMLElement {
  root.innerHTML = renderTemplate(tpl, { active });
  return root.querySelector<HTMLElement>('#adminMain')!;
}
