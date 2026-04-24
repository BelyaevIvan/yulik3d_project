import { renderTemplate } from '@/utils/template';
import { catalogTemplate } from './Catalog.template';
import { catalogApi } from '@/api/catalog';
import { productCardTemplate } from '@/components/ProductCard/ProductCard.template';
import { router } from '@/router/router';
import type { CategoryType, CategoryDTO, ItemCardDTO } from '@/api/types';
import './Catalog.scss';

const PAGE_SIZE = 20;

export class CatalogPage {
  private categories: CategoryDTO[] = [];
  constructor(
    private root: HTMLElement,
    private type: CategoryType,
    private query: URLSearchParams,
  ) {}

  async render(): Promise<void> {
    const baseUrl = this.type === 'figure' ? '/figurines' : '/models';
    const rootLabel = this.type === 'figure' ? 'Фигурки' : 'Макеты';
    const categoryId = this.query.get('category_id') || '';
    const subcategoryId = this.query.get('subcategory_id') || '';
    const q = this.query.get('q') || '';
    const sort = this.query.get('sort') || 'created_desc';
    const hasSale = this.query.get('has_sale') === 'true' ? true : undefined;
    const offset = parseInt(this.query.get('offset') || '0', 10) || 0;

    // Загружаем категории дерева для сайдбара
    if (this.categories.length === 0) {
      try {
        const cats = await catalogApi.listCategories(this.type, true);
        this.categories = cats.categories;
      } catch (e) { console.warn(e); }
    }

    // Loading state
    this.root.innerHTML = renderTemplate(catalogTemplate, {
      baseUrl, rootLabel, title: rootLabel,
      categories: [], loading: true, items: [], total: 0,
    });

    try {
      const res = await catalogApi.listItems({
        category_type: this.type,
        category_id: categoryId || undefined,
        subcategory_id: subcategoryId || undefined,
        q: q || undefined,
        sort,
        has_sale: hasSale,
        limit: PAGE_SIZE,
        offset,
      });

      // Title
      let title = rootLabel;
      let categoryLabel = '';
      let subcategoryLabel = '';
      const cat = this.categories.find((c) => c.id === categoryId);
      if (cat) { categoryLabel = cat.name; title = cat.name; }
      if (subcategoryId) {
        for (const c of this.categories) {
          const sub = (c.subcategories || []).find((s) => s.id === subcategoryId);
          if (sub) { subcategoryLabel = sub.name; categoryLabel = c.name; title = sub.name; break; }
        }
      }
      if (q) title = `Поиск: «${q}»`;

      const itemsHtml = res.items.map((it: ItemCardDTO) =>
        renderTemplate(productCardTemplate, it),
      ).join('');

      // Сайдбар-данные с активностью
      const cats = this.categories.map((c) => ({
        ...c,
        isActiveCat: c.id === categoryId,
        subcategories: (c.subcategories || []).map((s) => ({
          ...s, isActiveSub: s.id === subcategoryId,
        })),
      }));

      const totalPages = Math.max(1, Math.ceil(res.total / PAGE_SIZE));
      const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

      this.root.innerHTML = renderTemplate(catalogTemplate, {
        baseUrl, rootLabel, title,
        categories: cats,
        categoryLabel, subcategoryLabel,
        categoryId,
        isAllActive: !categoryId && !subcategoryId,
        loading: false,
        items: res.items,
        itemsHtml,
        total: res.total,
        sort,
        query: q,
        showPagination: totalPages > 1,
        hasPrev: offset > 0,
        hasNext: offset + PAGE_SIZE < res.total,
        currentPage, totalPages,
      });

      this.bindEvents(baseUrl, offset);
    } catch (e) {
      console.error('catalog:', e);
      this.root.innerHTML = renderTemplate(catalogTemplate, {
        baseUrl, rootLabel, title: rootLabel,
        categories: [], loading: false, items: [], total: 0,
      });
    }
  }

  private bindEvents(baseUrl: string, offset: number): void {
    const sortSel = this.root.querySelector<HTMLSelectElement>('#catalogSort');
    sortSel?.addEventListener('change', () => {
      const params = new URLSearchParams(this.query);
      params.set('sort', sortSel.value);
      params.delete('offset');
      router.navigate(baseUrl + '?' + params.toString());
    });

    this.root.querySelector('#prevPage')?.addEventListener('click', () => {
      const params = new URLSearchParams(this.query);
      const newOff = Math.max(0, offset - PAGE_SIZE);
      if (newOff === 0) params.delete('offset'); else params.set('offset', String(newOff));
      router.navigate(baseUrl + (params.toString() ? '?' + params.toString() : ''));
    });
    this.root.querySelector('#nextPage')?.addEventListener('click', () => {
      const params = new URLSearchParams(this.query);
      params.set('offset', String(offset + PAGE_SIZE));
      router.navigate(baseUrl + '?' + params.toString());
    });
  }
}
