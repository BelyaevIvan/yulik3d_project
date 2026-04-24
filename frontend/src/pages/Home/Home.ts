import { renderTemplate } from '@/utils/template';
import { homeTemplate } from './Home.template';
import { catalogApi } from '@/api/catalog';
import { productCardTemplate } from '@/components/ProductCard/ProductCard.template';
import type { ItemCardDTO } from '@/api/types';
import './Home.scss';

export class HomePage {
  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    this.root.innerHTML = renderTemplate(homeTemplate, {
      loadingFigurines: true, loadingModels: true,
      figurines: [], models: [],
      figurinesHtml: '', modelsHtml: '',
    });

    try {
      const [fig, mod] = await Promise.all([
        catalogApi.listItems({ category_type: 'figure', limit: 8 }),
        catalogApi.listItems({ category_type: 'other', limit: 8 }),
      ]);
      this.root.innerHTML = renderTemplate(homeTemplate, {
        loadingFigurines: false, loadingModels: false,
        figurines: fig.items, models: mod.items,
        figurinesHtml: this.cards(fig.items),
        modelsHtml: this.cards(mod.items),
      });
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
