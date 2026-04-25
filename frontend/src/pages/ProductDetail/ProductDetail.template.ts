export const productDetailTemplate = `
<div class="product">
  <nav class="breadcrumbs">
    <a href="/" data-link>Главная</a>
    <span>/</span>
    <a href="{{rootUrl}}" data-link>{{rootLabel}}</a>
    {{#if categoryName}}<span>/</span><a href="{{rootUrl}}?category_id={{categoryId}}" data-link>{{categoryName}}</a>{{/if}}
    <span>/</span><span>{{name}}</span>
  </nav>

  <div class="product__main">
    <div class="product__gallery">
      <div class="product__gallery-main">
        {{#if mainPicture}}
        <img src="{{mainPicture}}" alt="{{name}}" id="galleryMainImg" />
        {{else}}
        <div class="product__no-image">Нет изображения</div>
        {{/if}}
        {{#if (gt pictures.length 1)}}
        <button class="product__gallery-nav product__gallery-nav--prev" id="galleryPrev">‹</button>
        <button class="product__gallery-nav product__gallery-nav--next" id="galleryNext">›</button>
        {{/if}}
      </div>
      {{#if (gt pictures.length 1)}}
      <div class="product__thumbs">
        {{#each pictures}}
        <button class="product__thumb {{#if @first}}product__thumb--active{{/if}}" data-thumb="{{@index}}" data-url="{{url}}">
          <img src="{{url}}" alt="" />
        </button>
        {{/each}}
      </div>
      {{/if}}
    </div>

    <div class="product__info">
      <span class="product__articul">Артикул: {{articul}}</span>
      <h1 class="product__name">{{name}}</h1>

      <div class="product__price-block">
        <span class="product__price" id="livePrice">{{formatPrice final_price}}</span>
        {{#if (gt sale 0)}}
          <span class="product__price-old">{{formatPrice price}}</span>
          <span class="product__price-sale">−{{sale}}%</span>
        {{/if}}
      </div>

      <p class="product__availability {{#if hidden}}product__availability--no{{else}}product__availability--yes{{/if}}">
        {{#if hidden}}Не доступно к заказу{{else}}Доступно к заказу{{/if}}
      </p>

      {{#if options.length}}
      <div class="product__options">
        {{#each options}}
        <div class="product__option-group" data-type-id="{{type.id}}">
          <h4 class="product__option-label">{{type.label}}</h4>
          <div class="product__option-values">
            {{#each values}}
            <button class="product__option-value" data-option-id="{{id}}" data-extra="{{price}}">
              <span>{{value}}</span>
              {{#if (gt price 0)}}<span class="product__option-extra">+{{formatPrice price}}</span>{{/if}}
            </button>
            {{/each}}
          </div>
        </div>
        {{/each}}
      </div>
      {{/if}}

      <div class="product__actions">
        <button class="product__buy" id="addToCart" {{#if hidden}}disabled{{/if}}>
          {{#if hidden}}Недоступно{{else}}В корзину{{/if}}
        </button>
        <button class="product__fav" id="toggleFav" title="В избранное">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20.84 4.61a5.5 5.5 0 00-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 00-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 000-7.78z"/>
          </svg>
        </button>
      </div>
    </div>
  </div>

  <div class="product__details">
    <div class="product__details-block">
      <h3 class="product__details-title">Информация о товаре</h3>
      <div class="md-content">{{{descriptionInfoHtml}}}</div>
    </div>
    <div class="product__details-block">
      <h3 class="product__details-title">Особенности</h3>
      <div class="md-content">{{{descriptionOtherHtml}}}</div>
    </div>
  </div>

  <!-- Бонусные карточки -->
  <div class="product__bonus">
    <div class="product__bonus-card">
      <div class="product__bonus-icon">🎁</div>
      <h4>Подарок к заказу</h4>
      <p>К каждому заказу — фирменный мини-подарок от мастерской.</p>
    </div>
    <div class="product__bonus-card">
      <div class="product__bonus-icon">🚚</div>
      <h4>Доставка по России</h4>
      <p>Отправляем СДЭК и Почтой России. Самовывоз — без оплаты доставки.</p>
    </div>
    <div class="product__bonus-card">
      <div class="product__bonus-icon">💬</div>
      <h4>По всем вопросам — пишите</h4>
      <p>Telegram, ВКонтакте или почта — ответим в течение дня.</p>
    </div>
  </div>
</div>
`;
