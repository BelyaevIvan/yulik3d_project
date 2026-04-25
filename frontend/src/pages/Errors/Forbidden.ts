import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import './Errors.scss';

export class ForbiddenPage {
  constructor(private root: HTMLElement) {}
  render(): void {
    setPageMeta({ title: 'Нет доступа', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = `
      <div class="error-page">
        <div class="error-page__code">403</div>
        <h1 class="error-page__title">Нет доступа</h1>
        <p class="error-page__text">Эта страница доступна только администраторам или залогиненным пользователям. Если у вас должен быть доступ — проверьте, что вы вошли под нужным аккаунтом.</p>
        <div class="error-page__actions">
          <a href="/" data-link class="error-page__btn">На главную</a>
          <a href="/login" data-link class="error-page__btn--ghost">Войти</a>
        </div>
      </div>
    `;
  }
}
