import { renderTemplate } from '@/utils/template';
import { favoritesApi } from '@/api/favorites';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import { productCardTemplate } from '@/components/ProductCard/ProductCard.template';
import './Favorites.scss';

const tpl = `
<div class="favorites">
  <h1 class="favorites__title">Избранное</h1>
  {{#if loading}}
    <p class="favorites__empty">Загрузка...</p>
  {{else}}
    {{#if items.length}}
      <div class="product-grid">{{{itemsHtml}}}</div>
    {{else}}
      <div class="favorites__empty">
        <p>В избранном пока пусто.</p>
        <a href="/figurines" data-link>Перейти в каталог →</a>
      </div>
    {{/if}}
  {{/if}}
</div>
`;

export class FavoritesPage {
  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!authStore.isAuthed()) {
      router.navigate('/login?next=/favorites');
      return;
    }
    this.root.innerHTML = renderTemplate(tpl, { loading: true, items: [], itemsHtml: '' });
    try {
      const res = await favoritesApi.list(100, 0);
      const itemsHtml = res.items.map((it) => renderTemplate(productCardTemplate, it)).join('');
      this.root.innerHTML = renderTemplate(tpl, { loading: false, items: res.items, itemsHtml });
    } catch (e) {
      this.root.innerHTML = renderTemplate(tpl, { loading: false, items: [], itemsHtml: '' });
    }
  }
}
