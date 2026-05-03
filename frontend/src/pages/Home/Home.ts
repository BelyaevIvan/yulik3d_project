import { renderTemplate } from '@/utils/template';
import { homeTemplate } from './Home.template';
import { catalogApi } from '@/api/catalog';
import { productCardTemplate } from '@/components/ProductCard/ProductCard.template';
import { syncFavoriteButtons } from '@/utils/favoriteButtons';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import type { ItemCardDTO } from '@/api/types';
import './Home.scss';

export class HomePage {
  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    setPageMeta({
      description: 'Магазин 3D-печати YULIK3D. Фигурки персонажей из игр, фильмов и аниме на заказ. Декор, вазы, аксессуары. Полимерная смола, ручная работа, доставка по России.',
    });
    clearProductJsonLd();

    this.root.innerHTML = renderTemplate(homeTemplate, {
      loadingFigurines: true, loadingModels: true,
      figurines: [], models: [],
      figurinesHtml: '', modelsHtml: '',
    });

    try {
      // Главная: спец-эндпоинт /items/main отдаёт сначала закреплённые админом,
      // потом добивает свежими видимыми. Лимит 5 — один ряд на широком экране.
      const [fig, mod] = await Promise.all([
        catalogApi.mainPage('figure'),
        catalogApi.mainPage('other'),
      ]);
      this.root.innerHTML = renderTemplate(homeTemplate, {
        loadingFigurines: false, loadingModels: false,
        figurines: fig, models: mod,
        figurinesHtml: this.cards(fig),
        modelsHtml: this.cards(mod),
      });
      syncFavoriteButtons(this.root);
    } catch (e) {
      console.error('home:', e);
      this.root.innerHTML = renderTemplate(homeTemplate, {
        loadingFigurines: false, loadingModels: false,
        figurines: [], models: [], figurinesHtml: '', modelsHtml: '',
      });
    }
  }

  private cards(items: ItemCardDTO[]): string {
    return items.map((it) => renderTemplate(productCardTemplate, it)).join('');
  }
}
