export const catalogTemplate = `
<div class="catalog">
  <nav class="breadcrumbs">
    <a href="/" data-link>Главная</a>
    <span>/</span>
    <a href="{{baseUrl}}" data-link>{{rootLabel}}</a>
    {{#if categoryLabel}}<span>/</span><a href="{{baseUrl}}?category_id={{categoryId}}" data-link>{{categoryLabel}}</a>{{/if}}
    {{#if subcategoryLabel}}<span>/</span><span>{{subcategoryLabel}}</span>{{/if}}
  </nav>

  <h1 class="catalog__title">{{title}}</h1>

  <div class="catalog__layout">
    <aside class="catalog__sidebar">
      <h3 class="catalog__sidebar-title">Категории</h3>
      <a href="{{baseUrl}}" data-link class="catalog__sidebar-link {{#if isAllActive}}catalog__sidebar-link--active{{/if}}">Все товары</a>
      {{#each categories}}
        <div class="catalog__sidebar-group">
          <a href="{{../baseUrl}}?category_id={{id}}" data-link class="catalog__sidebar-cat {{#if isActiveCat}}catalog__sidebar-cat--active{{/if}}">{{name}}</a>
          {{#each subcategories}}
            <a href="{{../../baseUrl}}?subcategory_id={{id}}" data-link class="catalog__sidebar-sub {{#if isActiveSub}}catalog__sidebar-sub--active{{/if}}">{{name}}</a>
          {{/each}}
        </div>
      {{/each}}
    </aside>

    <main class="catalog__main">
      <div class="catalog__toolbar">
        <span class="catalog__count">Найдено: {{total}}</span>
        <select class="catalog__sort" id="catalogSort">
          <option value="created_desc"{{#if (eq sort "created_desc")}} selected{{/if}}>Сначала новые</option>
          <option value="price_asc"{{#if (eq sort "price_asc")}} selected{{/if}}>Цена ↑</option>
          <option value="price_desc"{{#if (eq sort "price_desc")}} selected{{/if}}>Цена ↓</option>
          <option value="name_asc"{{#if (eq sort "name_asc")}} selected{{/if}}>Имя А-Я</option>
          <option value="name_desc"{{#if (eq sort "name_desc")}} selected{{/if}}>Имя Я-А</option>
        </select>
      </div>

      {{#if loading}}
        <div class="catalog__loader">Загрузка...</div>
      {{else}}
        {{#if items.length}}
          <div class="product-grid">{{{itemsHtml}}}</div>
          {{#if showPagination}}
            <div class="catalog__pager">
              <button class="catalog__pager-btn" id="prevPage" {{#unless hasPrev}}disabled{{/unless}}>← Назад</button>
              <span class="catalog__pager-info">{{currentPage}} / {{totalPages}}</span>
              <button class="catalog__pager-btn" id="nextPage" {{#unless hasNext}}disabled{{/unless}}>Вперёд →</button>
            </div>
          {{/if}}
        {{else}}
          <div class="catalog__empty">
            <p>Ничего не найдено{{#if query}} по запросу «{{query}}»{{/if}}.</p>
            <a href="{{baseUrl}}" data-link>Сбросить фильтры →</a>
          </div>
        {{/if}}
      {{/if}}
    </main>
  </div>
</div>
`;
