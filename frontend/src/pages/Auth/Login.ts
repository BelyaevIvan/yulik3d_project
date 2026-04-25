import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import { renderTemplate } from '@/utils/template';
import { toast } from '@/components/Toast/Toast';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import './Auth.scss';

const tpl = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Вход</h1>
    <p class="auth__sub">Войдите в аккаунт, чтобы оформлять заказы</p>
    <form id="loginForm">
      <div id="loginErr" style="display:none" class="auth__error"></div>
      <div class="auth__field">
        <label class="auth__label" for="email">Email</label>
        <input class="auth__input" type="email" id="email" name="email" required autocomplete="email" />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="password">Пароль</label>
        <input class="auth__input" type="password" id="password" name="password" required autocomplete="current-password" />
      </div>
      <button type="submit" class="auth__submit" id="loginSubmit">Войти</button>
    </form>
    <div class="auth__alt">
      Нет аккаунта? <a href="/register" data-link>Зарегистрироваться</a>
    </div>
  </div>
</div>
`;

export class LoginPage {
  constructor(private root: HTMLElement, private query: URLSearchParams) {}

  render(): void {
    setPageMeta({ title: 'Вход', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = renderTemplate(tpl, {});
    const form = this.root.querySelector<HTMLFormElement>('#loginForm');
    const err = this.root.querySelector<HTMLElement>('#loginErr');
    const btn = this.root.querySelector<HTMLButtonElement>('#loginSubmit');
    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (err) err.style.display = 'none';
      const fd = new FormData(form);
      btn!.disabled = true;
      try {
        const u = await authApi.login({
          email: String(fd.get('email')),
          password: String(fd.get('password')),
        });
        authStore.setUser(u);
        toast.success('Добро пожаловать!');
        const next = this.query.get('next') || '/';
        router.navigate(next);
      } catch (e) {
        if (err) {
          err.textContent = e instanceof ApiError ? e.message : 'Ошибка входа';
          err.style.display = 'block';
        }
      } finally {
        btn!.disabled = false;
      }
    });
  }
}
