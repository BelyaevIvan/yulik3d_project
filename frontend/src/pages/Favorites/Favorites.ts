import { renderTemplate } from '@/utils/template';
import { favoritesApi } from '@/api/favorites';
import { authStore } from '@/store/auth';
import { favoritesStore } from '@/store/favorites';
import { router } from '@/router/router';
import { productCardTemplate } from '@/components/ProductCard/ProductCard.template';
import { syncFavoriteButtons } from '@/utils/favoriteButtons';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
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
  private unsub: (() => void) | null = null;

  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!authStore.isAuthed()) {
      router.navigate('/login?next=/favorites');
      return;
    }
    setPageMeta({ title: 'Избранное', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = renderTemplate(tpl, { loading: true, items: [], itemsHtml: '' });

    try {
      const res = await favoritesApi.list(100, 0);
      const itemsHtml = res.items.map((it) => renderTemplate(productCardTemplate, it)).join('');
      this.root.innerHTML = renderTemplate(tpl, { loading: false, items: res.items, itemsHtml });
      syncFavoriteButtons(this.root);

      // Перерисовываем при изменении избранного (удалили из карточки → список обновится).
      // Подписка должна быть активна ТОЛЬКО пока пользователь на /favorites.
      if (this.unsub) this.unsub();
      this.unsub = favoritesStore.subscribe(() => {
        // Если страница уже не /favorites — отписываемся, чтобы не сломать другие.
        if (!this.root.querySelector('.favorites')) {
          this.unsub?.();
          this.unsub = null;
          return;
        }
        this.root.querySelectorAll<HTMLElement>('[data-item-id]').forEach((card) => {
          const id = card.dataset.itemId!;
          if (!favoritesStore.has(id)) card.remove();
        });
        const remaining = this.root.querySelectorAll('[data-item-id]').length;
        if (remaining === 0) {
          this.root.innerHTML = renderTemplate(tpl, { loading: false, items: [], itemsHtml: '' });
        }
      });
    } catch (e) {
      this.root.innerHTML = renderTemplate(tpl, { loading: false, items: [], itemsHtml: '' });
    }
  }
}
