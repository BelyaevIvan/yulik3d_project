// Главная страница: hero-баннеры + featured товары.
export const homeTemplate = `
<div class="home">
  <!-- Hero-карточки (статика для дизайна) -->
  <section class="home__hero">
    <a href="/figurines" data-link class="home__hero-card home__hero-card--main">
      <div class="home__hero-bg" style="background-image: url('https://picsum.photos/seed/yulik3d-figurine/1200/600')"></div>
      <div class="home__hero-content">
        <span class="home__hero-tag">Новинки</span>
        <h2 class="home__hero-title">Фигурки на заказ</h2>
        <p class="home__hero-text">Любой персонаж — герой игры, фильма или ваш собственный портрет</p>
        <span class="home__hero-cta">Смотреть фигурки →</span>
      </div>
    </a>

    <div class="home__hero-side">
      <a href="/models" data-link class="home__hero-card">
        <div class="home__hero-bg" style="background-image: url('https://picsum.photos/seed/yulik3d-models/600/400')"></div>
        <div class="home__hero-content">
          <h3 class="home__hero-subtitle">Макеты и декор</h3>
          <span class="home__hero-cta">Перейти →</span>
        </div>
      </a>
      <a href="/figurines?has_sale=true" data-link class="home__hero-card home__hero-card--sale">
        <div class="home__hero-bg" style="background-image: url('https://picsum.photos/seed/yulik3d-sale/600/400')"></div>
        <div class="home__hero-content">
          <span class="home__hero-tag home__hero-tag--accent">Скидки</span>
          <h3 class="home__hero-subtitle">Товары со скидкой</h3>
          <span class="home__hero-cta">Подобрать →</span>
        </div>
      </a>
    </div>
  </section>

  <!-- Фигурки -->
  <section class="home__section">
    <div class="home__section-head">
      <h2 class="home__section-title">Фигурки</h2>
      <a href="/figurines" data-link class="home__section-all">Все фигурки →</a>
    </div>
    {{#if loadingFigurines}}
      <div class="home__loader">Загрузка...</div>
    {{else}}
      {{#if figurines.length}}
        <div class="product-grid product-grid--row">{{{figurinesHtml}}}</div>
      {{else}}
        <p class="home__empty">Пока нет товаров в этой категории.</p>
      {{/if}}
    {{/if}}
  </section>

  <!-- Макеты -->
  <section class="home__section">
    <div class="home__section-head">
      <h2 class="home__section-title">Макеты</h2>
      <a href="/models" data-link class="home__section-all">Все макеты →</a>
    </div>
    {{#if loadingModels}}
      <div class="home__loader">Загрузка...</div>
    {{else}}
      {{#if models.length}}
        <div class="product-grid product-grid--row">{{{modelsHtml}}}</div>
      {{else}}
        <p class="home__empty">Пока нет товаров в этой категории.</p>
      {{/if}}
    {{/if}}
  </section>
</div>
`;
