import './Errors.scss';

export class NotFoundPage {
  constructor(private root: HTMLElement) {}
  render(): void {
    this.root.innerHTML = `
      <div class="error-page">
        <div class="error-page__code">404</div>
        <h1 class="error-page__title">Страница не найдена</h1>
        <p class="error-page__text">Возможно, ссылка устарела или вы ввели её с ошибкой. Не страшно — вернитесь на главную или загляните в каталог.</p>
        <div class="error-page__actions">
          <a href="/" data-link class="error-page__btn">На главную</a>
          <a href="/figurines" data-link class="error-page__btn--ghost">В каталог</a>
        </div>
      </div>
    `;
  }
}
