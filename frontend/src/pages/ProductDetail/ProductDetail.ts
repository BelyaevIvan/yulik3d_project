import { renderTemplate } from '@/utils/template';
import { productDetailTemplate } from './ProductDetail.template';
import { catalogApi } from '@/api/catalog';
import { favoritesApi } from '@/api/favorites';
import { ApiError } from '@/api/client';
import { cartStore } from '@/store/cart';
import { authStore } from '@/store/auth';
import { favoritesStore } from '@/store/favorites';
import { router } from '@/router/router';
import { toast } from '@/components/Toast/Toast';
import { renderMarkdown } from '@/utils/markdown';
import type { ItemDetailDTO } from '@/api/types';
import './ProductDetail.scss';

export class ProductDetailPage {
  private item: ItemDetailDTO | null = null;
  private selectedOptions: Map<string, { id: string; price: number }> = new Map(); // typeId -> selection

  constructor(private root: HTMLElement, private id: string) {}

  async render(): Promise<void> {
    this.root.innerHTML = `<div class="product"><p style="text-align:center; padding:60px; color:#888;">Загрузка...</p></div>`;
    try {
      this.item = await catalogApi.getItem(this.id);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        router.replace('/404');
        return;
      }
      this.root.innerHTML = `<div class="product"><p style="text-align:center; padding:60px; color:#f44;">Ошибка загрузки товара.</p></div>`;
      return;
    }

    const it = this.item;
    const firstCat = it.subcategories[0]?.category;

    this.root.innerHTML = renderTemplate(productDetailTemplate, {
      ...it,
      mainPicture: it.pictures[0]?.url || null,
      rootUrl: firstCat?.type === 'figure' ? '/figurines' : '/models',
      rootLabel: firstCat?.type === 'figure' ? 'Фигурки' : (firstCat?.type === 'other' ? 'Макеты' : 'Каталог'),
      categoryName: firstCat?.name || '',
      categoryId: firstCat?.id || '',
      descriptionInfoHtml: renderMarkdown(it.description_info),
      descriptionOtherHtml: renderMarkdown(it.description_other),
    });

    this.bindEvents();
  }

  private bindEvents(): void {
    const it = this.item;
    if (!it) return;

    // Галерея
    const mainImg = this.root.querySelector<HTMLImageElement>('#galleryMainImg');
    const thumbs = Array.from(this.root.querySelectorAll<HTMLButtonElement>('.product__thumb'));
    let idx = 0;
    const setActive = (i: number) => {
      idx = (i + it.pictures.length) % it.pictures.length;
      thumbs.forEach((t, n) => t.classList.toggle('product__thumb--active', n === idx));
      if (mainImg && it.pictures[idx]) mainImg.src = it.pictures[idx].url;
    };
    thumbs.forEach((t) => t.addEventListener('click', () => setActive(parseInt(t.dataset.thumb || '0', 10))));
    this.root.querySelector('#galleryPrev')?.addEventListener('click', () => setActive(idx - 1));
    this.root.querySelector('#galleryNext')?.addEventListener('click', () => setActive(idx + 1));

    // Опции
    this.root.querySelectorAll<HTMLElement>('.product__option-group').forEach((group) => {
      const typeId = group.dataset.typeId || '';
      group.querySelectorAll<HTMLButtonElement>('.product__option-value').forEach((btn) => {
        btn.addEventListener('click', () => {
          group.querySelectorAll('.product__option-value').forEach((b) => b.classList.remove('product__option-value--active'));
          btn.classList.add('product__option-value--active');
          this.selectedOptions.set(typeId, {
            id: btn.dataset.optionId || '',
            price: parseInt(btn.dataset.extra || '0', 10),
          });
          this.updatePrice();
        });
      });
    });

    // Корзина
    this.root.querySelector('#addToCart')?.addEventListener('click', () => {
      if (!authStore.isAuthed()) {
        toast.info('Чтобы добавлять в корзину, войдите в аккаунт');
        router.navigate('/login?next=' + encodeURIComponent(`/product/${this.id}`));
        return;
      }
      // Проверка: если есть option-группы — нужно по одной из каждой
      if (it.options.length > this.selectedOptions.size) {
        toast.error('Выберите все опции');
        return;
      }
      const ids = Array.from(this.selectedOptions.values()).map((s) => s.id);
      cartStore.addFromDetail(it, ids, 1);
      toast.success('Товар добавлен в корзину');
    });

    // Избранное — синхронизуем начальное состояние со стором
    const favBtn = this.root.querySelector<HTMLButtonElement>('#toggleFav');
    if (favBtn) {
      const updateActive = () => favBtn.classList.toggle('product__fav--active', favoritesStore.has(it.id));
      updateActive();
      // Подписываемся, чтобы реагировать на внешние изменения
      favoritesStore.subscribe(updateActive);

      favBtn.addEventListener('click', async () => {
        if (!authStore.isAuthed()) {
          toast.info('Войдите, чтобы добавлять в избранное');
          router.navigate('/login?next=' + encodeURIComponent(`/product/${this.id}`));
          return;
        }
        try {
          if (favoritesStore.has(it.id)) {
            await favoritesApi.remove(it.id);
            favoritesStore.remove(it.id);
            toast.info('Удалено из избранного');
          } else {
            await favoritesApi.add(it.id);
            favoritesStore.add(it.id);
            toast.success('Добавлено в избранное');
          }
        } catch (e) {
          if (e instanceof ApiError) toast.error(e.message);
        }
      });
    }
  }

  private updatePrice(): void {
    if (!this.item) return;
    let extra = 0;
    this.selectedOptions.forEach((v) => extra += v.price);
    const finalPrice = this.item.final_price + extra;
    const el = this.root.querySelector<HTMLElement>('#livePrice');
    if (el) el.textContent = new Intl.NumberFormat('ru-RU').format(finalPrice) + ' ₽';
  }
}
