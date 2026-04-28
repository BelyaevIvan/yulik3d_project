import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { router } from '@/router/router';
import { renderTemplate } from '@/utils/template';
import { toast } from '@/components/Toast/Toast';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import './Auth.scss';

const tpl = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Новый пароль</h1>
    <p class="auth__sub">Придумайте новый пароль для своего аккаунта.</p>
    <form id="resetForm">
      <div id="resetErr" style="display:none" class="auth__error"></div>
      <div class="auth__field">
        <label class="auth__label" for="password">Новый пароль</label>
        <input class="auth__input" type="password" id="password" name="password" required minlength="8" autocomplete="new-password" />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="password2">Повторите пароль</label>
        <input class="auth__input" type="password" id="password2" name="password2" required minlength="8" autocomplete="new-password" />
      </div>
      <button type="submit" class="auth__submit" id="resetSubmit">Сменить пароль</button>
    </form>
  </div>
</div>
`;

const tplBadToken = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Ссылка не работает</h1>
    <p class="auth__sub">Кажется, ссылка устарела или была использована раньше. Запросите новую — старая ссылка действительна один раз и в течение часа.</p>
    <div class="auth__alt">
      <a href="/forgot-password" data-link>Запросить новую ссылку</a>
    </div>
  </div>
</div>
`;

export class ResetPasswordPage {
  constructor(private root: HTMLElement, private query: URLSearchParams) {}

  render(): void {
    setPageMeta({ title: 'Новый пароль', noindex: true });
    clearProductJsonLd();
    const token = this.query.get('token') || '';
    if (!token) {
      this.root.innerHTML = renderTemplate(tplBadToken, {});
      return;
    }
    this.root.innerHTML = renderTemplate(tpl, {});
    const form = this.root.querySelector<HTMLFormElement>('#resetForm');
    const err = this.root.querySelector<HTMLElement>('#resetErr');
    const btn = this.root.querySelector<HTMLButtonElement>('#resetSubmit');
    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (err) err.style.display = 'none';
      const fd = new FormData(form);
      const pwd = String(fd.get('password'));
      const pwd2 = String(fd.get('password2'));
      if (pwd !== pwd2) {
        if (err) { err.textContent = 'Пароли не совпадают'; err.style.display = 'block'; }
        return;
      }
      btn!.disabled = true;
      try {
        await authApi.passwordResetConfirm(token, pwd);
        toast.success('Пароль изменён. Войдите с новым паролем.');
        router.navigate('/login');
      } catch (e) {
        if (err) {
          err.textContent = e instanceof ApiError ? e.message : 'Ошибка смены пароля';
          err.style.display = 'block';
        }
      } finally {
        btn!.disabled = false;
      }
    });
  }
}
