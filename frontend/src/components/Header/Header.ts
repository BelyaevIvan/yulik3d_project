import { renderTemplate } from '@/utils/template';
import { headerTemplate, searchSuggestionTemplate } from './Header.template';
import { catalogApi } from '@/api/catalog';
import { authStore } from '@/store/auth';
import { cartStore } from '@/store/cart';
import { router } from '@/router/router';
import { toast } from '@/components/Toast/Toast';
import type { CategoryDTO, CategoryType, ItemCardDTO } from '@/api/types';
import './Header.scss';

interface NavGroup {
  label: string;
  listUrl: string;
  type: CategoryType;
  activeClass: string;
  categories: Array<CategoryDTO & { subcategories: Array<{ id: string; name: string }> }>;
}

export class Header {
  private root: HTMLElement;
  private categoriesByType: Record<CategoryType, CategoryDTO[]> = {
    figure: [],
    other: [],
  };
  private suggestionTimer: number | null = null;
  private currentNav: 'figurines' | 'models' | '' = '';

  constructor(root: HTMLElement) {
    this.root = root;
    authStore.subscribe(() => this.render());
    cartStore.subscribe(() => this.render());
  }

  setActive(nav: 'figurines' | 'models' | ''): void {
    this.currentNav = nav;
    this.render();
  }

  async init(): Promise<void> {
    try {
      const figurines = await catalogApi.listCategories('figure', true);
      const models = await catalogApi.listCategories('other', true);
      this.categoriesByType.figure = figurines.categories;
      this.categoriesByType.other = models.categories;
    } catch (e) {
      console.warn('header: failed to load categories', e);
    }
    this.render();
  }

  render(): void {
    const user = authStore.getUser();
    const groups: NavGroup[] = [
      {
        label: 'Фигурки',
        listUrl: '/figurines',
        type: 'figure',
        activeClass: this.currentNav === 'figurines' ? 'header__nav-link--active' : '',
        categories: this.categoriesByType.figure.map((c) => ({
          ...c, subcategories: c.subcategories || [],
        })),
      },
      {
        label: 'Макеты',
        listUrl: '/models',
        type: 'other',
        activeClass: this.currentNav === 'models' ? 'header__nav-link--active' : '',
        categories: this.categoriesByType.other.map((c) => ({
          ...c, subcategories: c.subcategories || [],
        })),
      },
    ];
    this.root.innerHTML = renderTemplate(headerTemplate, {
      navGroups: groups,
      cartCount: cartStore.count(),
      user,
      isAdmin: user?.role === 'admin',
    });
    this.bindEvents();
  }

  private bindEvents(): void {
    // Поиск
    const input = this.root.querySelector<HTMLInputElement>('#searchInput');
    const btn = this.root.querySelector<HTMLButtonElement>('#searchBtn');
    const sug = this.root.querySelector<HTMLElement>('#searchSuggestions');

    const submit = () => {
      const q = (input?.value || '').trim();
      if (!q) return;
      sug?.classList.remove('header__suggestions--visible');
      // Глобальный поиск — без фильтра по типу (фигурки + макеты)
      router.navigate('/search?q=' + encodeURIComponent(q));
    };

    btn?.addEventListener('click', submit);
    input?.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') submit();
      if (e.key === 'Escape') sug?.classList.remove('header__suggestions--visible');
    });

    input?.addEventListener('input', () => {
      const q = (input.value || '').trim();
      if (this.suggestionTimer) window.clearTimeout(this.suggestionTimer);
      if (!q) {
        sug?.classList.remove('header__suggestions--visible');
        return;
      }
      this.suggestionTimer = window.setTimeout(async () => {
        try {
          const res = await catalogApi.listItems({ q, limit: 6 });
          this.renderSuggestions(res.items);
        } catch (e) { /* silent */ }
      }, 250);
    });

    document.addEventListener('click', (e) => {
      if (!sug || !input) return;
      if (!(e.target as HTMLElement).closest('#searchWrapper')) {
        sug.classList.remove('header__suggestions--visible');
      }
    });

    // Burger
    const burger = this.root.querySelector('#headerBurger');
    const nav = this.root.querySelector('#headerNav');
    burger?.addEventListener('click', () => nav?.classList.toggle('header__nav--open'));

    // Тап по dropdown-родителю на мобиле — раскрытие
    this.root.querySelectorAll('.header__nav-item').forEach((item) => {
      item.querySelector('.header__nav-link')?.addEventListener('click', (e) => {
        if (window.innerWidth > 992) return;
        const dropdown = item.querySelector('.header__dropdown');
        if (!dropdown) return;
        const isOpen = item.classList.contains('header__nav-item--expanded');
        if (!isOpen) {
          e.preventDefault();
          item.classList.add('header__nav-item--expanded');
        }
      });
    });

    // User dropdown — клик-открытие
    const userBlock = this.root.querySelector('.header__user');
    userBlock?.querySelector('.header__user-btn')?.addEventListener('click', (e) => {
      e.stopPropagation();
      userBlock.classList.toggle('header__user--expanded');
    });
    document.addEventListener('click', () => userBlock?.classList.remove('header__user--expanded'));

    // Logout
    this.root.querySelector('#headerLogout')?.addEventListener('click', async (e) => {
      e.stopPropagation();
      await authStore.logout();
      toast.success('Вы вышли из аккаунта');
      router.navigate('/');
    });

    // «Войти» — передаём текущий путь как next, чтобы вернуть после логина
    this.root.querySelector('#headerLogin')?.addEventListener('click', (e) => {
      e.preventDefault();
      const cur = window.location.pathname + window.location.search;
      const next = cur === '/login' || cur === '/register' ? '/' : cur;
      router.navigate('/login?next=' + encodeURIComponent(next));
    });
  }

  private renderSuggestions(results: ItemCardDTO[]): void {
    const sug = this.root.querySelector<HTMLElement>('#searchSuggestions');
    if (!sug) return;
    if (!results.length) {
      sug.innerHTML = renderTemplate(searchSuggestionTemplate, { noResults: true });
    } else {
      sug.innerHTML = renderTemplate(searchSuggestionTemplate, { results });
    }
    sug.classList.add('header__suggestions--visible');
  }
}
