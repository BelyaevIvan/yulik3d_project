// Шапка с навигацией. Подкатегории фигурок и макетов подгружаются с бэка.
export const headerTemplate = `
<header class="header">
  <div class="header__inner">
    <a href="/" data-link class="header__logo">
      <img src="/logo.png" alt="YULIK3D" class="header__logo-img" />
      <span class="header__logo-text">YULIK3D</span>
    </a>

    <button class="header__burger" id="headerBurger" aria-label="Меню">
      <span></span><span></span><span></span>
    </button>

    <nav class="header__nav" id="headerNav">
      {{#each navGroups}}
      <div class="header__nav-item" data-dropdown>
        <a href="{{listUrl}}" data-link class="header__nav-link {{activeClass}}">{{label}}</a>
        {{#if categories.length}}
        <div class="header__dropdown">
          <div class="header__dropdown-inner">
            {{#each categories}}
            <div class="header__dropdown-col">
              <a href="{{../listUrl}}?category_id={{id}}" data-link class="header__dropdown-heading">{{name}}</a>
              <div class="header__dropdown-list">
                {{#each subcategories}}
                <a href="{{../../listUrl}}?subcategory_id={{id}}" data-link class="header__dropdown-link">{{name}}</a>
                {{/each}}
              </div>
              {{#if subcategories.length}}
              <a href="{{../listUrl}}?category_id={{id}}" data-link class="header__dropdown-all">Смотреть все →</a>
              {{/if}}
            </div>
            {{/each}}
          </div>
          <div class="header__dropdown-footer">
            <a href="{{listUrl}}" data-link class="header__dropdown-footer-link">Все категории →</a>
          </div>
        </div>
        {{/if}}
      </div>
      {{/each}}
    </nav>

    <div class="header__search" id="searchWrapper">
      <div class="header__search-inner">
        <span class="header__search-icon">🔍</span>
        <input type="text" class="header__search-input" placeholder="Поиск по каталогу" id="searchInput" autocomplete="off" />
      </div>
      <button class="header__search-btn" id="searchBtn">Найти</button>
      <div class="header__suggestions" id="searchSuggestions"></div>
    </div>

    <div class="header__actions">
      {{#if user}}
        <div class="header__user" data-dropdown>
          <button class="header__user-btn" aria-label="Меню пользователя">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 4-7 8-7s8 3 8 7"/></svg>
            <span class="header__user-name">{{user.full_name}}</span>
          </button>
          <div class="header__user-dropdown">
            <a href="/profile" data-link class="header__user-item">Профиль</a>
            <a href="/orders" data-link class="header__user-item">Мои заказы</a>
            <a href="/favorites" data-link class="header__user-item">Избранное</a>
            {{#if isAdmin}}<a href="/admin" data-link class="header__user-item header__user-item--admin">Админ-панель</a>{{/if}}
            <button class="header__user-item header__user-item--btn" id="headerLogout">Выйти</button>
          </div>
        </div>
      {{else}}
        <a href="/login" data-link class="header__login">Войти</a>
      {{/if}}
      <a href="/cart" data-link class="header__cart">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 01-8 0"/></svg>
        {{#if (gt cartCount 0)}}<span class="header__cart-badge">{{cartCount}}</span>{{/if}}
      </a>
    </div>
  </div>
</header>
`;

export const searchSuggestionTemplate = `
{{#if noResults}}
<div class="header__suggestion-item header__suggestion-item--empty">
  <span>Ничего не найдено</span>
</div>
{{else}}
{{#each results}}
<a href="/product/{{id}}" data-link class="header__suggestion-item">
  <span class="header__suggestion-text">{{name}}</span>
  <span class="header__suggestion-price">{{formatPrice final_price}}</span>
</a>
{{/each}}
{{/if}}
`;
