// Глобальные обработчики кнопок ♥ в карточках товаров.
// Подписываемся один раз в main.ts. Работает через event delegation:
// в любой ProductCard кнопка имеет data-fav-toggle="<itemID>".

import { favoritesApi } from '@/api/favorites';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { favoritesStore } from '@/store/favorites';
import { router } from '@/router/router';
import { toast } from '@/components/Toast/Toast';

/** Пометить активные кнопки ♥ исходя из текущего состояния FavoritesStore. */
export function syncFavoriteButtons(root: ParentNode = document): void {
  root.querySelectorAll<HTMLButtonElement>('[data-fav-toggle]').forEach((btn) => {
    const id = btn.dataset.favToggle!;
    btn.classList.toggle('product-card__fav--active', favoritesStore.has(id));
  });
}

/** Подключить глобальный обработчик кликов и подписку на изменения стора. */
export function initFavoriteButtons(): void {
  // Перерисовываем все кнопки при изменении стора
  favoritesStore.subscribe(() => syncFavoriteButtons(document));

  // Делегированный клик по любой кнопке ♥
  document.addEventListener('click', async (e) => {
    const btn = (e.target as HTMLElement).closest('[data-fav-toggle]') as HTMLButtonElement | null;
    if (!btn) return;
    e.preventDefault();
    e.stopPropagation();

    if (!authStore.isAuthed()) {
      toast.info('Войдите, чтобы добавлять в избранное');
      const cur = window.location.pathname + window.location.search;
      router.navigate('/login?next=' + encodeURIComponent(cur));
      return;
    }
    const id = btn.dataset.favToggle!;
    const isFav = favoritesStore.has(id);
    try {
      if (isFav) {
        await favoritesApi.remove(id);
        favoritesStore.remove(id);
        toast.info('Удалено из избранного');
      } else {
        await favoritesApi.add(id);
        favoritesStore.add(id);
        toast.success('Добавлено в избранное');
      }
    } catch (err) {
      if (err instanceof ApiError) toast.error(err.message);
    }
  });
}
