import './styles/global.scss';
import './components/ProductCard/ProductCard.scss';
import { router } from './router/router';
import { authStore } from './store/auth';
import { favoritesStore } from './store/favorites';
import { initFavoriteButtons } from './utils/favoriteButtons';
import { Header } from './components/Header/Header';
import { Footer } from './components/Footer/Footer';

import { HomePage } from './pages/Home/Home';
import { CatalogPage } from './pages/Catalog/Catalog';
import { ProductDetailPage } from './pages/ProductDetail/ProductDetail';
import { CartPage } from './pages/Cart/Cart';
import { LoginPage } from './pages/Auth/Login';
import { RegisterPage } from './pages/Auth/Register';
import { ProfilePage } from './pages/Profile/Profile';
import { FavoritesPage } from './pages/Favorites/Favorites';
import { MyOrdersPage } from './pages/Orders/MyOrders';
import { OrderDetailPage } from './pages/Orders/OrderDetail';
import { AdminItemsPage } from './pages/Admin/AdminItems';
import { AdminItemFormPage } from './pages/Admin/AdminItemForm';
import { AdminCategoriesPage } from './pages/Admin/AdminCategories';
import { AdminOptionTypesPage } from './pages/Admin/AdminOptionTypes';
import { AdminOrdersPage, AdminOrderDetailPage } from './pages/Admin/AdminOrders';
import { NotFoundPage } from './pages/Errors/NotFound';
import { ForbiddenPage } from './pages/Errors/Forbidden';

class App {
  private headerEl: HTMLElement;
  private contentEl: HTMLElement;
  private footerEl: HTMLElement;
  private header: Header;
  private footer: Footer;

  constructor() {
    const app = document.getElementById('app')!;

    this.headerEl = document.createElement('div');
    this.contentEl = document.createElement('main');
    this.contentEl.className = 'page-content';
    this.footerEl = document.createElement('div');

    app.appendChild(this.headerEl);
    app.appendChild(this.contentEl);
    app.appendChild(this.footerEl);

    this.header = new Header(this.headerEl);
    this.footer = new Footer(this.footerEl);
  }

  async start(): Promise<void> {
    // 1. Дёрнуть /me, чтобы знать кто авторизован
    await authStore.init();
    // 1а. Загрузить избранное (если юзер залогинен — стор подгрузит)
    if (authStore.isAuthed()) await favoritesStore.reload();
    // 1б. Глобальный обработчик ♥-кнопок в карточках
    initFavoriteButtons();
    // 2. Подгрузить категории для шапки
    await this.header.init();
    this.footer.render();
    // 3. Зарегистрировать роуты
    this.setupRoutes();
    router.start();
  }

  private setupRoutes(): void {
    const set = (nav: 'figurines' | 'models' | '') => this.header.setActive(nav);

    router.addRoute('/', () => { set(''); new HomePage(this.contentEl).render(); });

    router.addRoute('/figurines', (_, q) => { set('figurines'); new CatalogPage(this.contentEl, 'figure', q).render(); });
    router.addRoute('/models', (_, q) => { set('models'); new CatalogPage(this.contentEl, 'other', q).render(); });
    // Глобальный поиск — без фильтра по типу
    router.addRoute('/search', (_, q) => { set(''); new CatalogPage(this.contentEl, null, q).render(); });

    router.addRoute('/product/:id', (p) => { set(''); new ProductDetailPage(this.contentEl, p.id).render(); });

    router.addRoute('/cart', () => { set(''); new CartPage(this.contentEl).render(); });

    router.addRoute('/login', (_, q) => { set(''); new LoginPage(this.contentEl, q).render(); });
    router.addRoute('/register', (_, q) => { set(''); new RegisterPage(this.contentEl, q).render(); });

    router.addRoute('/profile', () => { set(''); new ProfilePage(this.contentEl).render(); });
    router.addRoute('/favorites', () => { set(''); new FavoritesPage(this.contentEl).render(); });
    router.addRoute('/orders', () => { set(''); new MyOrdersPage(this.contentEl).render(); });
    router.addRoute('/orders/:id', (p) => { set(''); new OrderDetailPage(this.contentEl, p.id).render(); });

    // Admin
    router.addRoute('/admin', () => { set(''); new AdminItemsPage(this.contentEl).render(); });
    router.addRoute('/admin/items/new', () => { set(''); new AdminItemFormPage(this.contentEl).render(); });
    router.addRoute('/admin/items/:id', (p) => { set(''); new AdminItemFormPage(this.contentEl, p.id).render(); });
    router.addRoute('/admin/categories', () => { set(''); new AdminCategoriesPage(this.contentEl).render(); });
    router.addRoute('/admin/option-types', () => { set(''); new AdminOptionTypesPage(this.contentEl).render(); });
    router.addRoute('/admin/orders', () => { set(''); new AdminOrdersPage(this.contentEl).render(); });
    router.addRoute('/admin/orders/:id', (p) => { set(''); new AdminOrderDetailPage(this.contentEl, p.id).render(); });

    // Error
    router.addRoute('/404', () => { set(''); new NotFoundPage(this.contentEl).render(); });
    router.addRoute('/403', () => { set(''); new ForbiddenPage(this.contentEl).render(); });

    // Fallback
    router.setNotFound(() => { set(''); new NotFoundPage(this.contentEl).render(); });
  }
}

document.addEventListener('DOMContentLoaded', () => {
  new App().start();
});
