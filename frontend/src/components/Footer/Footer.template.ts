export const footerTemplate = `
<footer class="footer">
  <div class="footer__inner">
    <div class="footer__brand">
      <img src="/logo.png" alt="YULIK3D" class="footer__logo" />
      <span class="footer__brand-text">Магазин 3D-печати YULIK3D</span>
      <p class="footer__about">Фигурки и макеты на заказ. Печать на профессиональном оборудовании.</p>
    </div>

    <div class="footer__col">
      <h4 class="footer__heading">Каталог</h4>
      <a href="/figurines" data-link class="footer__link">Фигурки</a>
      <a href="/models" data-link class="footer__link">Макеты</a>
      <a href="/figurines?has_sale=true" data-link class="footer__link">Со скидкой</a>
    </div>

    <div class="footer__col">
      <h4 class="footer__heading">Аккаунт</h4>
      <a href="/login" data-link class="footer__link">Войти</a>
      <a href="/register" data-link class="footer__link">Регистрация</a>
      <a href="/orders" data-link class="footer__link">Мои заказы</a>
      <a href="/favorites" data-link class="footer__link">Избранное</a>
    </div>

    <div class="footer__col">
      <h4 class="footer__heading">Связь с нами</h4>
      <a href="mailto:{{contact.email}}" class="footer__link">{{contact.email}}</a>
      <div class="footer__socials">
        <a href="{{contact.vk}}" target="_blank" rel="noopener" class="footer__social" title="ВКонтакте">VK</a>
        <a href="{{contact.telegram}}" target="_blank" rel="noopener" class="footer__social" title="Telegram">TG</a>
      </div>
    </div>
  </div>
  <div class="footer__bottom">
    <span>© {{year}} YULIK3D — все права защищены</span>
  </div>
</footer>
`;
