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
    <h1 class="auth__title">Регистрация</h1>
    <p class="auth__sub">Создайте аккаунт за минуту</p>
    <form id="regForm">
      <div id="regErr" style="display:none" class="auth__error"></div>
      <div class="auth__field">
        <label class="auth__label" for="email">Email</label>
        <input class="auth__input" type="email" id="email" name="email" required autocomplete="email" />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="full_name">Имя и фамилия</label>
        <input class="auth__input" type="text" id="full_name" name="full_name" required autocomplete="name" />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="phone">Телефон (необязательно)</label>
        <input class="auth__input" type="tel" id="phone" name="phone" autocomplete="tel" placeholder="+7 999 ..." />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="password">Пароль</label>
        <input class="auth__input" type="password" id="password" name="password" required minlength="8" autocomplete="new-password" />
      </div>
      <button type="submit" class="auth__submit" id="regSubmit">Создать аккаунт</button>
    </form>
    <div class="auth__alt">
      Уже есть аккаунт? <a href="/login" data-link>Войти</a>
    </div>
  </div>
</div>
`;

export class RegisterPage {
  constructor(private root: HTMLElement, private query: URLSearchParams) {}

  render(): void {
    setPageMeta({ title: 'Регистрация', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = renderTemplate(tpl, {});
    const form = this.root.querySelector<HTMLFormElement>('#regForm');
    const err = this.root.querySelector<HTMLElement>('#regErr');
    const btn = this.root.querySelector<HTMLButtonElement>('#regSubmit');
    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (err) err.style.display = 'none';
      const fd = new FormData(form);
      const phone = String(fd.get('phone') || '').trim();
      btn!.disabled = true;
      try {
        const u = await authApi.register({
          email: String(fd.get('email')),
          password: String(fd.get('password')),
          full_name: String(fd.get('full_name')),
          phone: phone || undefined,
        });
        authStore.setUser(u);
        toast.success('Аккаунт создан!');
        const next = this.query.get('next') || '/';
        router.navigate(next);
      } catch (e) {
        if (err) {
          err.textContent = e instanceof ApiError ? e.message : 'Ошибка регистрации';
          err.style.display = 'block';
        }
      } finally {
        btn!.disabled = false;
      }
    });
  }
}
