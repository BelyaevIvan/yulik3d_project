// Карточка товара (для каталога / избранного / админ-списка).
// Импорт SCSS — чтобы стили попали в бандл, когда template используется на любой странице.
import './ProductCard.scss';

export const productCardTemplate = `
<article class="product-card{{#if hidden}} product-card--hidden{{/if}}" data-item-id="{{id}}">
  <a href="/product/{{id}}" data-link class="product-card__link">
    <div class="product-card__image-wrap">
      {{#if primary_picture_url}}
      <img src="{{primary_picture_url}}" alt="{{name}}" class="product-card__image" loading="lazy" />
      {{else}}
      <div class="product-card__image product-card__image--placeholder">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>
      </div>
      {{/if}}
      {{#if hidden}}<span class="product-card__badge product-card__badge--hidden">Не доступно</span>{{/if}}
      {{#if (gt sale 0)}}<span class="product-card__badge product-card__badge--sale">−{{sale}}%</span>{{/if}}
    </div>
    <div class="product-card__body">
      {{#if category}}<span class="product-card__category">{{category.name}}</span>{{/if}}
      <h3 class="product-card__name">{{name}}</h3>
      <div class="product-card__price-block">
        <span class="product-card__price">{{formatPrice final_price}}</span>
        {{#if (gt sale 0)}}
          <span class="product-card__price-old">{{formatPrice price}}</span>
        {{/if}}
      </div>
    </div>
  </a>
  <button class="product-card__fav" data-fav-toggle="{{id}}" type="button" aria-label="В избранное" title="В избранное">
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M20.84 4.61a5.5 5.5 0 00-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 00-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 000-7.78z"/>
    </svg>
  </button>
</article>
`;
